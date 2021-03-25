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
	tempDir, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	conf.SetEnv(t, "FHIR_PAYLOAD_DIR", tempDir)
	conf.SetEnv(t, "FHIR_STAGING_DIR", tempDir)

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
	exMap := make(map[string]string)
	exMap["EnrollFlag1"] = "1"
	exMap["HCC_version"] = "V12"
	exMap["HCC_COL_1"] = "1"
	exMap["HCC_COL_2"] = "0"
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
			MetaKey:       2,
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

	// Add Data into repo
	_ = alrWorker.AlrRepository.AddAlr(ctx, cmsID, timestamp, alrs[:1])
	_ = alrWorker.AlrRepository.AddAlr(ctx, cmsID, timestamp2, alrs[1:2])

	r := postgres.NewRepository(db)
	acoUUID := uuid.NewRandom()
	aco := models.ACO{UUID: acoUUID, CMSID: &cmsID}
	err := r.CreateACO(ctx, aco)
	assert.NoError(t, err)
	job := models.Job{
		ACOID:             acoUUID,
		RequestURL:        "",
		Status:            models.JobStatusPending,
		TransactionTime:   time.Now(),
		JobCount:          1,
		CompletedJobCount: 0,
	}
	id, err := r.CreateJob(ctx, job)
	assert.NoError(t, err)

	// Create JobArgs
	jobArgs := models.JobAlrEnqueueArgs{
		ID:         id,
		CMSID:      cmsID,
		MBIs:       MBIs,
		LowerBound: timestamp,
		UpperBound: timestamp2,
	}
	enqueuer := queueing.NewEnqueuer()
	err = enqueuer.AddAlrJob(jobArgs, 100)
	assert.NoError(t, err)

	// Now start the workers...
	q := StartQue(log, 2)
	q.cloudWatchEnv = "dev"
	defer q.StopQue()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Job not completed in alloted time.")
			return
		default:
			currentJob, err := r.GetJobByID(ctx, id)
			assert.NoError(t, err)
			// don't wait for a job if it has a terminal status
			if isTerminalStatus(currentJob.Status) {
				return
			}
			log.Infof("Waiting on job to be completed. Current status %s.", currentJob.Status)
			time.Sleep(1 * time.Second)
		}
	}
}
