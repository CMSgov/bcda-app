package queueing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	workerRepo "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/bgentry/que-go"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
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

	conf.SetEnv(t, "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(t, "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(t, "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")

	// Ensure we do not clutter our working directory with any data
	tempDir1, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDir2, err := os.MkdirTemp("", "*")
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
	id, _ := safecast.ToInt(job.ID)
	jobArgs := models.JobEnqueueArgs{ID: id, ACOID: cmsID, BBBasePath: uuid.New()}

	enqueuer := NewEnqueuer()
	assert.NoError(t, enqueuer.AddJob(context.Background(), jobArgs, 1))

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
		JobCount: 2,
	}
	id, err := r.CreateJob(ctx, job)
	assert.NoError(t, err)

	// Create JobArgs
	k, _ := safecast.ToInt64(alr[0].MetaKey)
	jobArgs := models.JobAlrEnqueueArgs{
		ID:         id,
		CMSID:      cmsID,
		MetaKey:    k,
		MBIs:       []string{alr[0].BeneMBI},
		BBBasePath: "/v1/fhir",
		LowerBound: time.Time{},
		UpperBound: time.Time{},
	}

	key, _ := safecast.ToInt64(alr[0].MetaKey)
	jobArgs2 := models.JobAlrEnqueueArgs{
		ID:         id,
		CMSID:      cmsID,
		MetaKey:    key,
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
		ID:   testUtils.CryptoRandInt63(),
		Args: jobArgsJson,
	})
	assert.NoError(t, err)

	// Check job is in progress
	alrJob, err := r.GetJobByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, models.JobStatusInProgress, alrJob.Status)

	err = master.startAlrJob(&que.Job{
		ID:   testUtils.CryptoRandInt63(),
		Args: jobArgsJson2,
	})
	assert.NoError(t, err)

	// Check job is complete
	alrJob, err = r.GetJobByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, alrJob.Status)
}
