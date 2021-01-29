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
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	QUE_PROCESS_JOB = "ProcessJob"
)

// queue is responsible for retrieving jobs using the que client and
// transforming and delegating that work to the underlying worker
type queue struct {
	quePool           *que.WorkerPool
	pool              *pgx.ConnPool
	healthCheckCancel context.CancelFunc

	worker worker.Worker
}

// StartQue creates a que-go client and begins listening for items
// It returns immediately since all of the associated workers are started
// in separate goroutines.
func StartQue(queueDatabaseURL string, numWorkers int) *queue {
	// Allocate the queue in advance to supply the correct
	// in the workmap
	q := &queue{worker: worker.NewWorker(database.GetDbConnection())}

	cfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	q.pool, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   cfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Ensure that the connections are valid. Needed until we move to pgx v4
	ctx, cancel := context.WithCancel(context.Background())
	q.healthCheckCancel = cancel
	database.StartHealthCheck(ctx, q.pool, 10*time.Second)

	qc := que.NewClient(q.pool)
	wm := que.WorkMap{
		QUE_PROCESS_JOB: q.processJob,
	}

	q.quePool = que.NewWorkerPool(qc, wm, numWorkers)

	q.quePool.Start()

	return q
}

// StopQue cleans up any resources created
func (q *queue) StopQue() {
	q.healthCheckCancel()
	q.quePool.Shutdown()
	q.pool.Close()
}

func (q *queue) processJob(job *que.Job) error {
	ctx := context.Background()
	defer updateJobQueueCountCloudwatchMetric()

	var jobArgs models.JobEnqueueArgs
	err := json.Unmarshal(job.Args, &jobArgs)
	if err != nil {
		// ACK the job because retrying it won't help us be able to deserialize the data
		return nil
	}

	exportJob, err := q.worker.ValidateJob(ctx, jobArgs)
	if goerrors.Is(err, worker.ErrParentJobCancelled) {
		// ACK the job because we do not need to work on queue jobs associated with a cancelled parent job
		return nil
	} else if goerrors.Is(err, worker.ErrNoBasePathSet) {
		// Data is corrupted, we cannot work on this job.
		return nil
	} else if goerrors.Is(err, worker.ErrParentJobNotFound) {
		// Based on the current backoff delay (j.ErrorCount^4 + 3 seconds), this should've given
		// us plenty of headroom to ensure that the parent job will never be found.
		maxNotFoundRetries := int32(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))
		if job.ErrorCount >= maxNotFoundRetries {
			log.Errorf("No job found for ID: %d acoID: %s. Retries exhausted. Removing job from queue.", jobArgs.ID,
				jobArgs.ACOID)
			// By returning a nil error response, we're singaling to que-go to remove this job from the jobqueue.
			return nil
		}

		log.Warnf("No job found for ID %d acoID: %s. Will retry.", jobArgs.ID, jobArgs.ACOID)
		return errors.Wrap(repository.ErrJobNotFound, "could not retrieve job from database")
	} else if err != nil {
		return err
	}

	return q.worker.ProcessJob(ctx, *exportJob, jobArgs)
}

func getQueueJobCount() float64 {
	databaseURL := os.Getenv("QUEUE_DATABASE_URL")
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Error(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Error(pingErr)
	}
	defer db.Close()

	row := db.QueryRow(`select count(*) from que_jobs;`)

	var count int
	if err := row.Scan(&count); err != nil {
		log.Error(err)
	}

	return float64(count)
}

func updateJobQueueCountCloudwatchMetric() {

	// Update the Cloudwatch Metric for job queue count
	env := os.Getenv("DEPLOYMENT_TARGET")
	if env != "" {
		sampler, err := metrics.NewSampler("BCDA", "Count")
		if err != nil {
			fmt.Println("Warning: failed to create new metric sampler...")
		} else {
			err := sampler.PutSample("JobQueueCount", getQueueJobCount(), []metrics.Dimension{
				{Name: "Environment", Value: env},
			})
			if err != nil {
				log.Error(err)
			}
		}
	}
}
