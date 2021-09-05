package manager

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// queue is responsible for retrieving jobs using the que client and
// transforming and delegating that work to the underlying worker
type queue struct {
	// Resources associated with the underlying que client
	quePool *que.WorkerPool

	worker     worker.Worker
	repository repository.Repository
	log        logrus.FieldLogger
	queDB      *pgx.ConnPool

	cloudWatchEnv string
}

// Assignment List Report (ALR) shares the worker pool and "piggy-backs" off
// Beneficiary FHIR Data workflow. Instead of creating redundant functions and
// methods, masterQueue wraps both structs allows for sharing.
type masterQueue struct {
	*queue
	*alrQueue // This is defined in alr.go

	StagingDir string `conf:"FHIR_STAGING_DIR"`
	PayloadDir string `conf:"FHIR_PAYLOAD_DIR"`
	MaxRetry   int32  `conf:"BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES" conf_default:"3"`
}

func newMasterQueue(q *queue, qAlr *alrQueue) *masterQueue {
	mq := &masterQueue{
		queue:    q,
		alrQueue: qAlr,
	}

	if err := conf.Checkout(mq); err != nil {
		logrus.Fatal("Could not get data from conf for ALR.", err)
	}

	return mq
}

// StartQue creates a que-go client and begins listening for items
// It returns immediately since all of the associated workers are started
// in separate goroutines.
func StartQue(log logrus.FieldLogger, numWorkers int) *masterQueue {
	// Allocate the queue in advance to supply the correct
	// in the workmap
	mainDB := database.Connection
	q := &queue{
		worker:        worker.NewWorker(mainDB),
		repository:    postgres.NewRepository(mainDB),
		log:           log,
		queDB:         database.QueueConnection,
		cloudWatchEnv: conf.GetEnv("DEPLOYMENT_TARGET"),
	}
	// Same as above, but do one for ALR
	qAlr := &alrQueue{
		alrLog:    log,
		alrWorker: worker.NewAlrWorker(mainDB),
	}
	master := newMasterQueue(q, qAlr)

	qc := que.NewClient(q.queDB)
	wm := que.WorkMap{
		queueing.QUE_PROCESS_JOB: q.processJob,
		queueing.ALR_JOB:         master.startAlrJob, // ALR currently shares pool
	}

	q.quePool = que.NewWorkerPool(qc, wm, numWorkers)

	q.quePool.Start()

	return master
}

// StopQue cleans up any resources created
func (q *masterQueue) StopQue() {
	q.quePool.Shutdown()
}

func (q *queue) processJob(job *que.Job) error {
	ctx, cancel := context.WithCancel(context.Background())

	defer q.updateJobQueueCountCloudwatchMetric()
	defer cancel()

	var jobArgs models.JobEnqueueArgs
	err := json.Unmarshal(job.Args, &jobArgs)
	if err != nil {
		// ACK the job because retrying it won't help us be able to deserialize the data
		q.log.Warnf("Failed to deserialize job.Args '%s' %s. Removing queuejob from que.", job.Args, err)
		return nil
	}

	exportJob, err := q.worker.ValidateJob(ctx, jobArgs)
	if goerrors.Is(err, worker.ErrParentJobCancelled) {
		// ACK the job because we do not need to work on queue jobs associated with a cancelled parent job
		q.log.Warnf("queJob %d associated with a cancelled parent Job %d. Removing queuejob from que.", job.ID, jobArgs.ID)
		return nil
	} else if goerrors.Is(err, worker.ErrNoBasePathSet) {
		// Data is corrupted, we cannot work on this job.
		q.log.Warnf("Job %d does not contain valid base path. Removing queuejob from que.", jobArgs.ID)
		return nil
	} else if goerrors.Is(err, worker.ErrParentJobNotFound) {
		// Based on the current backoff delay (j.ErrorCount^4 + 3 seconds), this should've given
		// us plenty of headroom to ensure that the parent job will never be found.
		maxNotFoundRetries := int32(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))
		if job.ErrorCount >= maxNotFoundRetries {
			q.log.Errorf("No job found for ID: %d acoID: %s. Retries exhausted. Removing job from queue.", jobArgs.ID,
				jobArgs.ACOID)
			// By returning a nil error response, we're singaling to que-go to remove this job from the jobqueue.
			return nil
		}

		q.log.Warnf("No job found for ID: %d acoID: %s. Will retry.", jobArgs.ID, jobArgs.ACOID)
		return errors.Wrap(repository.ErrJobNotFound, "could not retrieve job from database")
	} else if err != nil {
		return errors.Wrap(err, "failed to validate job")
	}

	// start a goroutine that will periodically check the status of the parent job
	go checkIfCancelled(ctx, q.repository, cancel, uint(jobArgs.ID), 15)

	if err := q.worker.ProcessJob(ctx, *exportJob, jobArgs); err != nil {
		return errors.Wrap(err, "failed to process job")
	}

	return nil
}

func (q *queue) updateJobQueueCountCloudwatchMetric() {

	// Update the Cloudwatch Metric for job queue count
	if q.cloudWatchEnv != "" {
		sampler, err := metrics.NewSampler("BCDA", "Count")
		if err != nil {
			fmt.Println("Warning: failed to create new metric sampler...")
		} else {
			err := sampler.PutSample("JobQueueCount", q.getQueueJobCount(), []metrics.Dimension{
				{Name: "Environment", Value: q.cloudWatchEnv},
			})
			if err != nil {
				q.log.Error(err)
			}
		}
	}
}

func (q *queue) getQueueJobCount() float64 {
	row := q.queDB.QueryRow(`select count(*) from que_jobs;`)

	var count int
	if err := row.Scan(&count); err != nil {
		q.log.Error(err)
	}

	return float64(count)
}
