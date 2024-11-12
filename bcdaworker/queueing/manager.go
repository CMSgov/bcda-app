package queueing

import (
	"context"

	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
)

// queue is responsible for retrieving jobs using the que client and
// transforming and delegating that work to the underlying worker
// type queue struct {
// 	worker worker.Worker

// 	// Resources associated with the underlying que client
// 	quePool *que.WorkerPool
// 	queDB   *pgx.ConnPool

// 	// Resources associated with river client
// 	client *river.Client[pgxv5.Tx]
// 	// riverDB *pgxpool.Pool

// 	ctx           context.Context
// 	repository    repository.Repository
// 	log           logrus.FieldLogger
// 	cloudWatchEnv string
// }

// type queue[T models.JobEnqueueArgs | manager.RiverJobArgs, C *que.Client | *river.Client[pgxv5.Tx]] struct {
type queue struct {
	worker worker.Worker

	// Resources associated with the underlying Que client
	quePool *que.WorkerPool
	queDB   *pgx.ConnPool

	// Resources associated with River
	client *river.Client[pgxv5.Tx]

	ctx           context.Context
	repository    repository.Repository
	log           logrus.FieldLogger
	cloudWatchEnv string
}

// func (q *queue) validateJob() error {
// 	if goerrors.Is(err, worker.ErrParentJobCancelled) {
// 		// ACK the job because we do not need to work on queue jobs associated with a cancelled parent job
// 		logger.Warnf("queJob %d associated with a cancelled parent Job %d. Removing queuejob from que.", queJob.ID, jobArgs.ID)
// 		return nil
// 	} else if goerrors.Is(err, worker.ErrParentJobFailed) {
// 		// ACK the job because we do not need to work on queue jobs associated with a failed parent job
// 		logger.Warnf("queJob %d associated with a failed parent Job %d. Removing queuejob from que.", queJob.ID, jobArgs.ID)
// 		return nil
// 	} else if goerrors.Is(err, worker.ErrNoBasePathSet) {
// 		// Data is corrupted, we cannot work on this job.
// 		logger.Warnf("Job %d does not contain valid base path. Removing queuejob from que.", jobArgs.ID)
// 		return nil
// 	} else if goerrors.Is(err, worker.ErrParentJobNotFound) {
// 		// Based on the current backoff delay (j.ErrorCount^4 + 3 seconds), this should've given
// 		// us plenty of headroom to ensure that the parent job will never be found.
// 		maxNotFoundRetries, err := safecast.ToInt32(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))
// 		if err != nil {
// 			logger.Errorf("Failed to convert BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES to int32. Defaulting to 3. Error: %s", err)
// 			return nil
// 		}

// 		if queJob.ErrorCount >= maxNotFoundRetries {
// 			logger.Errorf("No job found for ID: %d acoID: %s. Retries exhausted. Removing job from queue.", jobArgs.ID,
// 				jobArgs.ACOID)
// 			// By returning a nil error response, we're signaling to que-go to remove this job from the job queue.
// 			return nil
// 		}

// 		logger.Warnf("No job found for ID: %d acoID: %s. Will retry.", jobArgs.ID, jobArgs.ACOID)
// 		return errors.Wrap(repository.ErrJobNotFound, "could not retrieve job from database")
// 	} else if goerrors.Is(err, worker.ErrQueJobProcessed) {
// 		logger.Warnf("Queue job (que_jobs.id) %d already processed for job.id %d. Checking completion status and removing queuejob from que.", queJob.ID, id)

// 		_, err := worker.CheckJobCompleteAndCleanup(ctx, q.repository, id)
// 		if err != nil {
// 			return errors.Wrap(err, fmt.Sprintf("Error checking job completion & cleanup for job id %d", id))
// 		}
// 		return nil
// 	} else if err != nil {
// 		err := errors.Wrap(err, "failed to validate job")
// 		logger.Error(err)
// 		return err
// 	}

// 	return nil
// }

// func (q *queue) updateJobQueueCountCloudwatchMetric() {

// 	// Update the Cloudwatch Metric for job queue count
// 	if q.cloudWatchEnv != "" {
// 		sampler, err := metrics.NewSampler("BCDA", "Count")
// 		if err != nil {
// 			fmt.Println("Warning: failed to create new metric sampler...")
// 		} else {
// 			err := sampler.PutSample("JobQueueCount", q.getQueueJobCount(), []metrics.Dimension{
// 				{Name: "Environment", Value: q.cloudWatchEnv},
// 			})
// 			if err != nil {
// 				q.log.Error(err)
// 			}
// 		}
// 	}
// }

// func (q *queue) getQueueJobCount() float64 {
// 	row := q.queDB.QueryRow(`select count(*) from que_jobs;`)

// 	var count int
// 	if err := row.Scan(&count); err != nil {
// 		q.log.Error(err)
// 	}

// 	return float64(count)
// }
