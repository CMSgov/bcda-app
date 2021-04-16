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
)

// logHook allows us to retrieve the messages emitted by the logging instance
var log = logrus.New()
var logHook = test.NewLocal(log)

// TestProcessJob acts as an end-to-end verification of the process.
// It uses the postgres/que-go backed implementations
func TestProcessJob(t *testing.T) {
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

	q := StartQue(log, 1)
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
			log.Infof("Waiting on job to be completed. Current status %s.", job.Status)
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
	queue := &queue{log: log}
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

			queue := &queue{worker: worker, log: log}

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

	// Create synthetic Data
	// TODO: Replace this with Martin's testing strategy from #4239
	// To do this, this requires bringing in worker que migrations

	// Create the key value pair
	exMap := make(map[string]string)
	exMap["EnrollFlag1"] = "1"
	exMap["HCC_version"] = "V12"
	exMap["HCC_COL_1"] = "1"
	exMap["HCC_COL_2"] = "0"
	// ACO
	cmsID := "A1234"
	MBIs := []string{"abd123abd01", "abd123abd02"}
	timestamp := time.Now()
	timestamp2 := timestamp.Add(time.Hour * 24)
	dob1, _ := time.Parse("01/02/2006", "01/20/1950")
	dob2, _ := time.Parse("01/02/2006", "04/15/1950")
	alrs := []models.Alr{
		{
			ID:            1, // These are set manually for testing
			MetaKey:       1, // PostgreSQL should automatically make these
			BeneMBI:       MBIs[0],
			BeneHIC:       "1q2w3e4r5t6y",
			BeneFirstName: "John",
			BeneLastName:  "Smith",
			BeneSex:       "1",
			BeneDOB:       dob1,
			BeneDOD:       time.Time{},
			KeyValue:      exMap,
		},
		{
			ID:            2,
			MetaKey:       1,
			BeneMBI:       MBIs[1],
			BeneHIC:       "0p9o8i7u6y5t",
			BeneFirstName: "Melissa",
			BeneLastName:  "Jones",
			BeneSex:       "2",
			BeneDOB:       dob2,
			BeneDOD:       time.Time{},
			KeyValue:      exMap,
		},
	}

	ctx := context.Background()

	// Add synthetic Data above into DB
	err := alrWorker.AlrRepository.AddAlr(ctx, cmsID, timestamp, alrs)
	assert.NoError(t, err)

	r := postgres.NewRepository(db)

	// Create data needed for models.JobAlrEnQueueArgs & models.Jobs
	// These are two data struct involved in ALr jobs
	acoUUID := uuid.NewRandom()
	aco := models.ACO{UUID: acoUUID, CMSID: &cmsID}
	// Add the ACO into aco table
	err = r.CreateACO(ctx, aco)
	assert.NoError(t, err)
	job := models.Job{
		ACOID:           acoUUID,
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
		QueueID:    1,
		CMSID:      cmsID,
		MBIs:       MBIs[:1],
		LowerBound: timestamp,
		UpperBound: timestamp2,
	}
	jobArgs2 := models.JobAlrEnqueueArgs{
		ID:         id,
		QueueID:    2,
		CMSID:      cmsID,
		MBIs:       MBIs[1:],
		LowerBound: timestamp,
		UpperBound: timestamp2,
	}

	// marshall jobs
	jobArgsJson, err := json.Marshal(jobArgs)
	assert.NoError(t, err)
	jobArgs2Json, err := json.Marshal(jobArgs2)
	assert.NoError(t, err)

	q := &queue{
		worker:        worker.NewWorker(db),
		repository:    workerRepo.NewRepository(db),
		log:           log,
		queDB:         database.QueueConnection,
		cloudWatchEnv: conf.GetEnv("DEPLOYMENT_TARGET"),
	}
	// Same as above, but do one for ALR
	qAlr := &alrQueue{
		alrLog:    log,
		alrWorker: worker.NewAlrWorker(db),
	}
	master := newMasterQueue(q, qAlr)

	// Since the worker is tested by BFD, it is not tested here
	// and we jump straight to the work
	err = master.startAlrJob(&que.Job{
		Args: jobArgsJson,
	})
	assert.NoError(t, err)
	err = master.startAlrJob(&que.Job{
		Args: jobArgs2Json,
	})
	assert.NoError(t, err)

	// Check job is complete
	alrJob, err := r.GetJobByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, alrJob.Status, models.JobStatusCompleted)
}
