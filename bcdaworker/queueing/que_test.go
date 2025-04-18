package queueing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	workerRepo "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/bgentry/que-go"
	"github.com/ccoveille/go-safecast"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// logHook allows us to retrieve the messages emitted by the logging instance
var logger = logrus.New()
var logHook = test.NewLocal(logger)

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
