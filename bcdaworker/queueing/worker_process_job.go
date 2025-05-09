package queueing

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"
)

type JobWorker struct {
	river.WorkerDefaults[worker_types.JobEnqueueArgs]
}

func (w *JobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.JobEnqueueArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobID, err := safecast.ToInt64(rjob.Args.ID)
	if err != nil {
		return err
	}

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	ctx, _ = log.SetCtxLogger(ctx, "job_id", jobID)
	ctx, logger := log.SetCtxLogger(ctx, "transaction_id", rjob.Args.TransactionID)

	// TODO: use pgxv5 when available
	mainDB := database.Connection
	workerInstance := worker.NewWorker(mainDB)
	repo := postgres.NewRepository(mainDB)

	defer updateJobQueueCountCloudwatchMetric(mainDB, logger)

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
