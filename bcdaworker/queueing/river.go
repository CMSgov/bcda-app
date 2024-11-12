package queueing

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/sirupsen/logrus"
)

// This is a duplicate of models.JobEnqueueArgs but with json
// type RiverJobArgs struct {
// 	ID              int       `json:"id"`
// 	ACOID           string    `json:"aco_id"`
// 	CMSID           string    `json:"cms_id"`
// 	BeneficiaryIDs  []string  `json:"beneficiary_ids"`
// 	ResourceType    string    `json:"resource_type"`
// 	Since           string    `json:"since"`
// 	TransactionID   string    `json:"transaction_id"`
// 	TransactionTime time.Time `json:"transaction_time"`
// 	BBBasePath      string    `json:"bb_base_path"`
// 	ClaimsWindow    struct {
// 		LowerBound time.Time `json:"lower_bound"`
// 		UpperBound time.Time `json:"upper_bound"`
// 	} `json:"claims_window"`
// 	DataType string `json:"data_type"`
// }

// func (jobargs RiverJobArgs) Kind() string {
// 	return models.QUE_PROCESS_JOB
// }

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
		// client:     riverClient,
		worker:        worker.NewWorker(mainDB),
		repository:    postgres.NewRepository(mainDB),
		log:           log,
		cloudWatchEnv: conf.GetEnv("DEPLOYMENT_TARGET"),
	}

	return q
}

func (q queue) StopRiver() {
	if err := q.client.Stop(q.ctx); err != nil {
		panic(err)
	}
}

type JobWorker struct {
	river.WorkerDefaults[models.JobEnqueueArgs]
}

func (w *JobWorker) Work(ctx context.Context, job *river.Job[models.JobEnqueueArgs]) error {
	// fmt.Printf("River Work! job: %+v", job)
	// ctx, cancel := context.WithCancel(ctx)

	// // defer updateJobQueueCountCloudwatchMetric()
	// defer cancel()

	// ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	// ctx, _ = log.SetCtxLogger(ctx, "job_id", job.Args.ID)
	// ctx, logger := log.SetCtxLogger(ctx, "transaction_id", job.Args.TransactionID)

	// id, err := safecast.ToUint(job.Args.ID)
	// if err != nil {
	// 	return err
	// }

	// // exportJob, err := q.worker.ValidateJob(ctx, id, job.Args)
	// if goerrors.Is(err, worker.ErrParentJobCancelled) {
	// 	// ACK the job because we do not need to work on queue jobs associated with a cancelled parent job
	// 	logger.Warnf("queJob %d associated with a cancelled parent Job %d. Removing queuejob from que.", queJob.ID, jobArgs.ID)
	// 	return nil
	// } else if goerrors.Is(err, worker.ErrParentJobFailed) {
	// 	// ACK the job because we do not need to work on queue jobs associated with a failed parent job
	// 	logger.Warnf("queJob %d associated with a failed parent Job %d. Removing queuejob from que.", queJob.ID, jobArgs.ID)
	// 	return nil
	// } else if goerrors.Is(err, worker.ErrNoBasePathSet) {
	// 	// Data is corrupted, we cannot work on this job.
	// 	logger.Warnf("Job %d does not contain valid base path. Removing queuejob from que.", jobArgs.ID)
	// 	return nil
	// } else if goerrors.Is(err, worker.ErrParentJobNotFound) {
	// 	// Based on the current backoff delay (j.ErrorCount^4 + 3 seconds), this should've given
	// 	// us plenty of headroom to ensure that the parent job will never be found.
	// 	maxNotFoundRetries, err := safecast.ToInt32(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))
	// 	if err != nil {
	// 		logger.Errorf("Failed to convert BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES to int32. Defaulting to 3. Error: %s", err)
	// 		return nil
	// 	}

	// 	if queJob.ErrorCount >= maxNotFoundRetries {
	// 		logger.Errorf("No job found for ID: %d acoID: %s. Retries exhausted. Removing job from queue.", jobArgs.ID,
	// 			jobArgs.ACOID)
	// 		// By returning a nil error response, we're signaling to que-go to remove this job from the job queue.
	// 		return nil
	// 	}

	// 	logger.Warnf("No job found for ID: %d acoID: %s. Will retry.", jobArgs.ID, jobArgs.ACOID)
	// 	return errors.Wrap(repository.ErrJobNotFound, "could not retrieve job from database")
	// } else if goerrors.Is(err, worker.ErrQueJobProcessed) {
	// 	logger.Warnf("Queue job (que_jobs.id) %d already processed for job.id %d. Checking completion status and removing queuejob from que.", queJob.ID, id)

	// 	_, err := worker.CheckJobCompleteAndCleanup(ctx, q.repository, id)
	// 	if err != nil {
	// 		return errors.Wrap(err, fmt.Sprintf("Error checking job completion & cleanup for job id %d", id))
	// 	}
	// 	return nil
	// } else if err != nil {
	// 	err := errors.Wrap(err, "failed to validate job")
	// 	logger.Error(err)
	// 	return err
	// }

	// // start a goroutine that will periodically check the status of the parent job
	// go checkIfCancelled(ctx, q.repository, cancel, id, 15)

	// if err := q.worker.ProcessJob(ctx, queJob.ID, *exportJob, jobArgs); err != nil {
	// 	err := errors.Wrap(err, "failed to process job")
	// 	logger.Error(err)
	// 	return err
	// }

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
