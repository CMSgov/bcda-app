package queueing

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
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
		// TODO: whats an appropriate timeout?  default is 1m
		// JobTimeout: 10 * time.Minute,
		JobTimeout: -1, // default for river is 1m, que-go had no timeout, mimicking que-go for now
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
		ctx:        context.Background(),
		client:     riverClient,
		worker:     worker.NewWorker(mainDB),
		repository: postgres.NewRepository(mainDB),
		log:        log,
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	ctx, _ = log.SetCtxLogger(ctx, "job_id", job.Args.ID)
	ctx, logger := log.SetCtxLogger(ctx, "transaction_id", job.Args.TransactionID)

	// TODO: use pgxv5 when available
	mainDB := database.Connection
	workerInstance := worker.NewWorker(mainDB)
	repo := postgres.NewRepository(mainDB)

	defer updateJobQueueCountCloudwatchMetric(mainDB, logger)

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

	// start a goroutine that will periodically check the status of the parent job
	go checkIfCancelled(ctx, repo, cancel, jobID, 15)

	if err := workerInstance.ProcessJob(ctx, jobID, *exportJob, job.Args); err != nil {
		err := errors.Wrap(err, "failed to process job")
		logger.Error(err)
		return err
	}

	return nil
}

// func logger(logger *logrus.Logger, outputFile string,
// 	application, environment string) logrus.FieldLogger {

// 	if outputFile != "" {
// 		// #nosec G302 -- 0640 permissions required for Splunk ingestion
// 		if file, err := os.OpenFile(filepath.Clean(outputFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
// 			logger.SetOutput(file)
// 		} else {
// 			logger.Infof("Failed to open output file %s. Will use stderr. %s",
// 				outputFile, err.Error())
// 		}
// 	}
// 	// Disable the HTML escape so we get the raw URLs
// 	logger.SetFormatter(&logrus.JSONFormatter{
// 		DisableHTMLEscape: true,
// 		TimestampFormat:   time.RFC3339Nano,
// 	})
// 	logger.SetReportCaller(true)

// 	return logger.WithFields(logrus.Fields{
// 		"application": application,
// 		"environment": environment,
// 		"version":     constants.Version})
// }

// TODO: once we remove que library and upgrade to pgx5 we can move the below functions into manager
// Update the AWS Cloudwatch Metric for job queue count
func updateJobQueueCountCloudwatchMetric(db *sql.DB, log logrus.FieldLogger) {
	cloudWatchEnv := conf.GetEnv("DEPLOYMENT_TARGET")
	if cloudWatchEnv != "" {
		sampler, err := metrics.NewSampler("BCDA", "Count")
		if err != nil {
			fmt.Println("Warning: failed to create new metric sampler...")
		} else {
			err := sampler.PutSample("JobQueueCount", getQueueJobCount(db, log), []metrics.Dimension{
				{Name: "Environment", Value: cloudWatchEnv},
			})
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func getQueueJobCount(db *sql.DB, log logrus.FieldLogger) float64 {
	row := db.QueryRow(`select count(*) from que_jobs;`)

	var count int
	if err := row.Scan(&count); err != nil {
		log.Error(err)
	}

	return float64(count)
}
