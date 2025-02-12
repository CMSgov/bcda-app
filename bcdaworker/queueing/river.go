package queueing

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/robfig/cron/v3"
	sloglogrus "github.com/samber/slog-logrus"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

type CleanupJobArgs struct {
}

func (args CleanupJobArgs) Kind() string {
	return "CleanupJob"
}

type CleanupJobWorker struct {
	river.WorkerDefaults[CleanupJobArgs]
	cleanupJob      func(time.Time, models.JobStatus, models.JobStatus, ...string) error
	archiveExpiring func(time.Time) error
}

type Notifier interface {
	PostMessageContext(context.Context, string, ...slack.MsgOption) (string, string, error)
}

// TODO: better dependency injection (db, worker, logger).  Waiting for pgxv5 upgrade
func StartRiver(numWorkers int) *queue {
	workers := river.NewWorkers()
	river.AddWorker(workers, &JobWorker{})
	river.AddWorker(workers, &CleanupJobWorker{})

	schedule, err := cron.ParseStandard("0 11,23 * * *")

	if err != nil {
		panic("Invalid cron schedule")
	}

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			schedule,
			func() (river.JobArgs, *river.InsertOpts) {
				return CleanupJobArgs{}, &river.InsertOpts{
					UniqueOpts: river.UniqueOpts{
						ByArgs: true,
					},
				}
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}

	riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: numWorkers},
		},
		// TODO: whats an appropriate timeout?
		JobTimeout:   -1, // default for river is 1m, que-go had no timeout, mimicking que-go for now
		Logger:       getSlogLogger(),
		Workers:      workers,
		PeriodicJobs: periodicJobs,
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

func (w *CleanupJobWorker) Work(ctx context.Context, rjob *river.Job[CleanupJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	_, logger := log.SetCtxLogger(ctx, "transaction_id", uuid.New())

	cutoff := getCutOffTime()
	archiveDir := conf.GetEnv("FHIR_ARCHIVE_DIR")
	stagingDir := conf.GetEnv("FHIR_STAGING_DIR")
	payloadDir := conf.GetEnv("FHIR_PAYLOAD_DIR")

	logger.Info("Starting Archive and Clean Job Data for environment (using os): %s.", os.Getenv("ENV"))
	logger.Info("Starting Archive and Clean Job Data for environment (using conf): %s.", conf.GetEnv("ENV"))
	logger.Info("Using directories: %s | %s | %s", archiveDir, stagingDir, payloadDir)

	params, err := getAWSParams()

	if err != nil {
		logger.Error("Unable to extract Slack Token from parameter store: %+v", err)
		return err
	}

	slackClient := slack.New(params)

	_, _, err = slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started Archive and Clean Job Data for environment: %s.", os.Getenv("ENV")), false),
	)

	if err != nil {
		logger.Error("Error sending notifier start message: %+v", err)
	}

	// Cleanup archived jobs: remove job directory and files from archive and update job status to Expired
	if err := w.cleanupJob(cutoff, models.JobStatusArchived, models.JobStatusExpired, archiveDir, stagingDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupArchArg)))

		_, _, err := slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: %s job in %s env.", constants.CleanupArchArg, os.Getenv("ENV")), false),
		)
		if err != nil {
			logger.Error("Error sending notifier failure message: %+v", err)
		}

		return err
	}

	// Cleanup failed jobs: remove job directory and files from failed jobs and update job status to FailedExpired
	if err := w.cleanupJob(cutoff, models.JobStatusFailed, models.JobStatusFailedExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupFailedArg)))

		_, _, err := slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: %s job in %s env.", constants.CleanupFailedArg, os.Getenv("ENV")), false),
		)
		if err != nil {
			logger.Error("Error sending notifier failure message: %+v", err)
		}

		return err
	}

	// Cleanup cancelled jobs: remove job directory and files from cancelled jobs and update job status to CancelledExpired
	if err := w.cleanupJob(cutoff, models.JobStatusCancelled, models.JobStatusCancelledExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupCancelledArg)))

		_, _, err := slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: %s job in %s env.", constants.CleanupCancelledArg, os.Getenv("ENV")), false),
		)
		if err != nil {
			logger.Error("Error sending notifier failure message: %+v", err)
		}

		return err
	}

	// Archive expiring jobs: update job statuses and move files to an inaccessible location
	if err := w.archiveExpiring(cutoff); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.ArchiveJobFiles)))

		_, _, err := slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: %s job in %s env.", constants.ArchiveJobFiles, os.Getenv("ENV")), false),
		)
		if err != nil {
			logger.Error("Error sending notifier failure message: %+v", err)
		}

		return err
	}

	_, _, err = slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("SUCCESS: Archive and Clean Job Data for %s environment.", os.Getenv("ENV")), false),
	)

	if err != nil {
		logger.Error("Error sending notifier success message: %+v", err)
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
	row := db.QueryRow(`SELECT COUNT(*) FROM river_job WHERE state NOT IN ('completed', 'cancelled', 'discarded');`)

	var count int
	if err := row.Scan(&count); err != nil {
		log.Error(err)
	}

	return float64(count)
}

func getCutOffTime() time.Time {
	cutoff := time.Now().Add(-time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24)))
	return cutoff
}

func getAWSParams() (string, error) {
	env := conf.GetEnv("ENV")

	if env == "local" {
		return conf.GetEnv("workflow-alerts"), nil
	}

	bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return "", err
	}

	slackToken, err := bcdaaws.GetParameter(bcdaSession, "/slack/token/workflow-alerts")
	if err != nil {
		return slackToken, err
	}

	return slackToken, nil
}
