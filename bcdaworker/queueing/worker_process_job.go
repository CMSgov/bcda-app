package queueing

import (
	"context"
	"database/sql"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
)

type JobWorker struct {
	river.WorkerDefaults[worker_types.JobEnqueueArgs]
	db *sql.DB
}

// previously this was set to -1 which translates to no timeout, 30m seems like plenty of time
// to handle a subjob, but could do with some future adjustment if needed
func (w *JobWorker) Timeout(*river.Job[worker_types.JobEnqueueArgs]) time.Duration {
	minutes := utils.GetEnvInt("PROCESS_JOB_TIMEOUT_MINUTES", 30)
	return time.Duration(minutes) * time.Minute
}

func (w *JobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.JobEnqueueArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			jobID, err := safecast.ToInt64(rjob.Args.ID)
			if err != nil {
				return err
			}

			ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
			ctx, logger := log.SetLoggerFields(ctx, logrus.Fields{"transaction_id": rjob.Args.TransactionID, "job_id": jobID})

			// TODO: use pgxv5 when available
			mainDB := w.db
			workerInstance := worker.NewWorker(mainDB)
			repo := postgres.NewRepository(mainDB)

			defer updateJobQueueCountCloudwatchMetric(ctx, mainDB, logger)

			exportJob, err, ackJob := validateJob(ctx, ValidateJobConfig{
				WorkerInstance: workerInstance,
				Logger:         logger,
				Repository:     repo,
				JobID:          jobID,
				QJobID:         rjob.ID,
				Args:           rjob.Args,
				ErrorCount:     len(rjob.Errors),
			})
			if ackJob {
				// End logic here, basically acknowledge and return which will remove it from the queue.
				return nil
			}
			// Return error when we want to mark a job as having errored out, which will mark it to be retried
			if err != nil {
				return err
			}

			// start a goroutine that will periodically check the status of the parent job
			go checkIfCancelled(ctx, repo, cancel, jobID, 15)

			if err := workerInstance.ProcessJob(ctx, rjob.ID, *exportJob, rjob.Args); err != nil {
				err := errors.Wrap(err, "failed to process job")
				logger.Error(err)
				return err
			}

			return nil
		}
	}
}
