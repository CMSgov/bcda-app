package manager

import (
	"context"

	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"

	pgxv5 "github.com/jackc/pgx/v5"
)

// queue is responsible for retrieving jobs using the que client and
// transforming and delegating that work to the underlying worker
type queue struct {
	// Resources associated with the underlying que client
	quePool *que.WorkerPool
	worker  worker.Worker

	// Resources associated with river client
	client *river.Client[pgxv5.Tx]
	ctx    context.Context

	repository repository.Repository
	log        logrus.FieldLogger
	queDB      *pgx.ConnPool

	cloudWatchEnv string
}

// Assignment List Report (ALR) shares the worker pool and "piggy-backs" off
// Beneficiary FHIR Data workflow. Instead of creating redundant functions and
// methods, masterQueue wraps both structs allows for sharing.
type MasterQueue struct {
	*queue
	*alrQueue // This is defined in alr.go

	StagingDir string `conf:"FHIR_STAGING_DIR"`
	PayloadDir string `conf:"FHIR_PAYLOAD_DIR"`
	MaxRetry   int32  `conf:"BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES" conf_default:"3"`
}

func newMasterQueue(q *queue, qAlr *alrQueue) *MasterQueue {
	mq := &MasterQueue{
		queue:    q,
		alrQueue: qAlr,
	}

	if err := conf.Checkout(mq); err != nil {
		logrus.Fatal("Could not get data from conf for ALR.", err)
	}

	return mq
}
