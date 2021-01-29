package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// queue is responsible for retrieving jobs using the que client and
// transforming and delegating that work to the underlying worker
type queue struct {
	// Resources associated with the underlying que client
	quePool           *que.WorkerPool
	pool              *pgx.ConnPool
	healthCheckCancel context.CancelFunc

	worker worker.Worker
	log    *logrus.Logger
	queDB  *sql.DB

	cloudWatchEnv string
}

// StartQue creates a que-go client and begins listening for items
// It returns immediately since all of the associated workers are started
// in separate goroutines.
func StartQue(log *logrus.Logger, queueDatabaseURL string, numWorkers int) *queue {
	db, err := sql.Open("postgres", queueDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Allocate the queue in advance to supply the correct
	// in the workmap
	q := &queue{
		worker:        worker.NewWorker(database.GetDbConnection()),
		log:           log,
		queDB:         db,
		cloudWatchEnv: os.Getenv("DEPLOYMENT_TARGET"),
	}

	cfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		q.log.Fatal(err)
	}

	q.pool, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   cfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		q.log.Fatal(err)
	}

	// Ensure that the connections are valid. Needed until we move to pgx v4
	ctx, cancel := context.WithCancel(context.Background())
	q.healthCheckCancel = cancel
	database.StartHealthCheck(ctx, q.pool, 10*time.Second)

	qc := que.NewClient(q.pool)
	wm := que.WorkMap{
		queueing.QUE_PROCESS_JOB: q.processJob,
	}

	q.quePool = que.NewWorkerPool(qc, wm, numWorkers)

	q.quePool.Start()

	return q
}

// StopQue cleans up any resources created
func (q *queue) StopQue() {
	q.healthCheckCancel()
	if err := q.queDB.Close(); err != nil {
		q.log.Warnf("Failed to close connection to queue database %s", err)
	}
	q.pool.Close()
	q.quePool.Shutdown()
}

func (q *queue) processJob(job *que.Job) error {
	ctx := context.Background()
	defer q.updateJobQueueCountCloudwatchMetric()

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
