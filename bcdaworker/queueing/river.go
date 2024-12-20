package queueing

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
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
	sloglogrus "github.com/samber/slog-logrus"
	"github.com/sirupsen/logrus"
)

// TODO: better dependency injection (db, worker, logger).  Waiting for pgxv5 upgrade
func StartRiver(numWorkers int) *queue {
	workers := river.NewWorkers()
	river.AddWorker(workers, &JobWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: numWorkers},
		},
		// TODO: whats an appropriate timeout?
		JobTimeout: -1, // default for river is 1m, que-go had no timeout, mimicking que-go for now
		Logger:     getSlogLogger(),
		Workers:    workers,
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
	}

	return q
}

// River requires a slog.Logger for logging, this function converts logrus to slog
// Much of this function is pulled from logger.go
func getSlogLogger() *slog.Logger {
	logrusLogger := logrus.New()

	outputFile := conf.GetEnv("BCDA_WORKER_ERROR_LOG")
	if outputFile != "" {
		// #nosec G302 -- 0640 permissions required for Splunk ingestion
		if file, err := os.OpenFile(filepath.Clean(outputFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
			logrusLogger.SetOutput(file)
		} else {
			logrusLogger.Infof("Failed to open output file %s. Will use stderr. %s",
				outputFile, err.Error())
		}
	}
	// Disable the HTML escape so we get the raw URLs
	logrusLogger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	logrusLogger.SetReportCaller(true)

	logrusLogger.WithFields(logrus.Fields{
		"application": "worker",
		"environment": conf.GetEnv("DEPLOYMENT_TARGET"),
		"version":     constants.Version,
	})

	return slog.New(sloglogrus.Option{Logger: logrusLogger}.NewLogrusHandler())
}

func (q queue) StopRiver() {
	if err := q.client.Stop(q.ctx); err != nil {
		panic(err)
	}
}

type JobWorker struct {
	river.WorkerDefaults[models.JobEnqueueArgs]
}

func (w *JobWorker) Work(ctx context.Context, rjob *river.Job[models.JobEnqueueArgs]) error {
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
	row := db.QueryRow(`select count(*) from river_job;`)

	var count int
	if err := row.Scan(&count); err != nil {
		log.Error(err)
	}

	return float64(count)
}
