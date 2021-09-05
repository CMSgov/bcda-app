package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	workerRepo "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// logHook allows us to retrieve the messages emitted by the logging instance
var logger = logrus.New()
var logHook = test.NewLocal(logger)

// TestProcessJob acts as an end-to-end verification of the process.
// It uses the postgres/que-go backed implementations
func TestProcessJob(t *testing.T) {
	// Set up the logger since we're using the real client
	client.SetLogger(logger)

	// Reset our environment once we've finished with the test
	defer func(payload, staging string) {
		conf.SetEnv(t, "FHIR_PAYLOAD_DIR", payload)
		conf.SetEnv(t, "FHIR_STAGING_DIR", staging)
	}(conf.GetEnv("FHIR_PAYLOAD_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))

	defer func(cert, key, ca string) {
		conf.SetEnv(t, "BB_CLIENT_CERT_FILE", cert)
		conf.SetEnv(t, "BB_CLIENT_KEY_FILE", key)
		conf.SetEnv(t, "BB_CLIENT_CA_FILE", ca)
	}(conf.GetEnv("BB_CLIENT_CERT_FILE"), conf.GetEnv("BB_CLIENT_KEY_FILE"), conf.GetEnv("BB_CLIENT_CA_FILE"))

	conf.SetEnv(t, "BB_CLIENT_CERT_FILE", "../../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(t, "BB_CLIENT_KEY_FILE", "../../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(t, "BB_CLIENT_CA_FILE", "../../../shared_files/localhost.crt")

	// Ensure we do not clutter our working directory with any data
	tempDir1, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDir2, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	conf.SetEnv(t, "FHIR_PAYLOAD_DIR", tempDir1)
	conf.SetEnv(t, "FHIR_STAGING_DIR", tempDir2)

	db := database.Connection

	cmsID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{UUID: uuid.NewRandom(), CMSID: &cmsID}
	postgrestest.CreateACO(t, db, aco)
	job := models.Job{ACOID: aco.UUID, Status: models.JobStatusPending}
	postgrestest.CreateJobs(t, db, &job)

	defer postgrestest.DeleteACO(t, db, aco.UUID)

	q := StartQue(logger, 1)
	q.cloudWatchEnv = "dev"
	defer q.StopQue()
	// Since the jobArgs does not have any beneIDs, the job should complete almost immediately
	jobArgs := models.JobEnqueueArgs{ID: int(job.ID), ACOID: cmsID, BBBasePath: uuid.New()}

	enqueuer := queueing.NewEnqueuer()
	assert.NoError(t, enqueuer.AddJob(jobArgs, 1))

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Job not completed in alloted time.")
			return
		default:
			currentJob := postgrestest.GetJobByID(t, db, job.ID)
			// don't wait for a job if it has a terminal status
			if isTerminalStatus(currentJob.Status) {
				return
			}
			logger.Infof("Waiting on job to be completed. Current status %s.", job.Status)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func isTerminalStatus(status models.JobStatus) bool {
	switch status {
	case models.JobStatusCompleted,
		models.JobStatusCancelled,
		models.JobStatusFailed:
		return true
	}
	return false
}

func TestProcessJobInvalidArgs(t *testing.T) {
	job := &que.Job{Args: []byte("{invalid_json")}
	queue := &queue{log: logger}
	assert.NoError(t, queue.processJob(job),
		"No error since invalid job data should not be retried")
	entry := logHook.LastEntry()
	assert.NotNil(t, entry)
	assert.Contains(t, entry.Message,
		fmt.Sprintf("Failed to deserialize job.Args '%s'", job.Args))
}

func TestProcessJobFailedValidation(t *testing.T) {
	tests := []struct {
		name        string
		validateErr error
		expectedErr error
		expLogMsg   string
	}{
		{"ParentJobCancelled", worker.ErrParentJobCancelled, nil, `^queJob \d+ associated with a cancelled parent Job`},
		{"NoBasePath", worker.ErrNoBasePathSet, nil, `^Job \d+ does not contain valid base path`},
		{"NoParentJob", worker.ErrParentJobNotFound, repository.ErrJobNotFound, `^No job found for ID: \d+ acoID.*Will retry`},
		{"NoParentJobRetriesExceeded", worker.ErrParentJobNotFound, nil, `No job found for ID: \d+ acoID.*Retries exhausted`},
		{"OtherError", fmt.Errorf("some other error"), fmt.Errorf("some other error"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			worker := &worker.MockWorker{}
			defer worker.AssertExpectations(t)

			queue := &queue{worker: worker, log: logger}

			job := models.Job{ID: uint(rand.Int31())}
			jobArgs := models.JobEnqueueArgs{ID: int(job.ID), ACOID: uuid.New()}

			var queJob que.Job
			queJob.Args, err = json.Marshal(jobArgs)
			assert.NoError(t, err)

			// Set the error count to max to ensure that we've exceeded the retries
			if tt.name == "NoParentJobRetriesExceeded" {
				queJob.ErrorCount = rand.Int31()
			}

			worker.On("ValidateJob", testUtils.CtxMatcher, jobArgs).Return(nil, tt.validateErr)

			err = queue.processJob(&queJob)
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			}

			if tt.expLogMsg != "" {
				assert.Regexp(t, regexp.MustCompile(tt.expLogMsg), logHook.LastEntry().Message)
			}
		})
	}

}

// Test ALR startAlrjob
func TestStartAlrJob(t *testing.T) {
	// Set up data based on testfixtures
	db := database.Connection
	alrWorker := worker.NewAlrWorker(db)
	ctx := context.Background()
	cmsID := "A9994"

	r := postgres.NewRepository(db)

	// Retreive ACO info
	aco, err := r.GetACOByCMSID(ctx, cmsID)
	assert.NoError(t, err)

	mbis, err := r.GetAlrMBIs(ctx, *aco.CMSID)
	assert.NoError(t, err)

	alr, err := alrWorker.GetAlr(ctx, mbis.Metakey, mbis.MBIS)
	assert.NoError(t, err)

	// Add the ACO into aco table
	job := models.Job{
		ACOID:           aco.UUID,
		RequestURL:      "",
		Status:          models.JobStatusPending,
		TransactionTime: time.Now(),
		// JobCount is partitioned automatically, but it is done manually here
		JobCount:          2,
		CompletedJobCount: 0,
	}
	id, err := r.CreateJob(ctx, job)
	assert.NoError(t, err)

	// Create JobArgs
	jobArgs := models.JobAlrEnqueueArgs{
		ID:         id,
		CMSID:      cmsID,
		MBIs:       []string{alr[0].BeneMBI},
		BBBasePath: "/v1/fhir",
		LowerBound: time.Time{},
		UpperBound: time.Time{},
	}
	jobArgs2 := models.JobAlrEnqueueArgs{
		ID:         id,
		CMSID:      cmsID,
		BBBasePath: "/v1/fhir",
		MBIs:       []string{alr[1].BeneMBI},
		LowerBound: time.Time{},
		UpperBound: time.Time{},
	}

	// marshal jobs
	jobArgsJson, err := json.Marshal(jobArgs)
	assert.NoError(t, err)
	jobArgsJson2, err := json.Marshal(jobArgs2)
	assert.NoError(t, err)

	q := &queue{
		worker:        worker.NewWorker(db),
		repository:    workerRepo.NewRepository(db),
		log:           logger,
		queDB:         database.QueueConnection,
		cloudWatchEnv: conf.GetEnv("DEPLOYMENT_TARGET"),
	}
	// Same as above, but do one for ALR
	qAlr := &alrQueue{
		alrLog:    logger,
		alrWorker: alrWorker,
	}
	master := newMasterQueue(q, qAlr)

	// Since the worker is tested by BFD, it is not tested here
	// and we jump straight to the work
	err = master.startAlrJob(&que.Job{
		Args: jobArgsJson,
	})
	assert.NoError(t, err)
	err = master.startAlrJob(&que.Job{
		Args: jobArgsJson2,
	})
	assert.NoError(t, err)

	// Check job is complete
	alrJob, err := r.GetJobByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, alrJob.Status)
}

// Test alr cancel logic
func TestAlrJobCancel(t *testing.T) {
	q := masterQueue{
		queue:      &queue{
			repository:    nil,
		},
		StagingDir: "",
		PayloadDir: "",
		MaxRetry:   0,
	}

	mockRepo := repository.MockRepository{}
	mockRepo.On("GetJobByID", testUtils.CtxMatcher, mock.Anything).Return(
		&models.Job{
			Status: models.JobStatusInProgress,
		},
		nil,
	).Once()
	mockRepo.On("GetJobByID", testUtils.CtxMatcher, mock.Anything).Return(
		&models.Job{
			Status: models.JobStatusCancelled,
		},
		nil,
	)
	q.repository = &mockRepo

	ctx, cancel := context.WithCancel(context.Background())
	jobs := models.JobAlrEnqueueArgs{}

	// In produation we wait 15 second intervals, for test we do 1
	go checkIfCancelled(ctx, q.repository, cancel, jobs.ID, 1)

	// Check if the context has been cancelled
	var cnt uint8
	var success bool

	Outer:
	for {
		select {
		case <-time.After(time.Second):
			if cnt > 10 {
				break Outer
			}
			cnt++
		case <-ctx.Done():
			success = true
			break Outer
		}
	}

	assert.True(t, success)

}
