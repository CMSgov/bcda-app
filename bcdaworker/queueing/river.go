package queueing

import (
	"context"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/sirupsen/logrus"
)

func StartRiver(log logrus.FieldLogger, numWorkers int) *queue {
	workers := river.NewWorkers()
	river.AddWorker(workers, &JobWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Connection), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: numWorkers},
		},
		// TODO: https://pkg.go.dev/github.com/darvaza-proxy/slog/handlers/logrus
		// Logger:  log,
		Workers: workers,
	})
	if err != nil {
		panic(err)
	}

	if err := riverClient.Start(context.Background()); err != nil {
		panic(err)
	}

	mainDB := database.Connection
	q := &queue{
		ctx:           context.Background(),
		client:        riverClient,
		worker:        worker.NewWorker(mainDB),
		repository:    postgres.NewRepository(mainDB),
		log:           log,
		cloudWatchEnv: conf.GetEnv("DEPLOYMENT_TARGET"),
	}

	return q
}

func (q queue) StopRiver() {
	fmt.Printf("---STOP RIVER: %+v, %+v\n", q, q.client)
	if err := q.client.Stop(q.ctx); err != nil {
		panic(err)
	}
}

type JobWorker struct {
	river.WorkerDefaults[models.JobEnqueueArgs]
}

func (w *JobWorker) Work(ctx context.Context, job *river.Job[models.JobEnqueueArgs]) error {
	fmt.Printf("---Is River processjobbing: %+v\n", job)
	ctx, cancel := context.WithCancel(ctx)

	// TODO
	// defer updateJobQueueCountCloudwatchMetric()
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	ctx, _ = log.SetCtxLogger(ctx, "job_id", job.Args.ID)
	ctx, logger := log.SetCtxLogger(ctx, "transaction_id", job.Args.TransactionID)

	// TODO: is this ok per worker?
	mainDB := database.Connection
	workerInstance := worker.NewWorker(mainDB)
	repo := postgres.NewRepository(mainDB)

	jobID, err := safecast.ToInt64(job.Args.ID)
	if err != nil {
		return err
	}

	exportJob, err, ackJob := validateJob(ctx, ValidateJobConfig{
		WorkerInstance: workerInstance,
		Logger:         logger,
		Repository:     repo,
		JobID:          jobID,
		QJobID:         job.ID,
		Args:           job.Args,
		ErrorCount:     len(job.Errors),
	})
	if ackJob {
		// End logic here, basically acknowledge and return which will remove it from the queue.
		return nil
	}
	// Return error when we want to mark a job as having errored out, which will mark it to be retried
	if err != nil {
		return err
	}

	fmt.Printf("---exportJob: %+v\n", exportJob)
	fmt.Printf("---exportJob error: %+v\n", err)

	// start a goroutine that will periodically check the status of the parent job
	go checkIfCancelled(ctx, repo, cancel, jobID, 15)

	if err := workerInstance.ProcessJob(ctx, jobID, *exportJob, job.Args); err != nil {
		err := errors.Wrap(err, "failed to process job")
		logger.Error(err)
		return err
	}

	return nil
}

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
