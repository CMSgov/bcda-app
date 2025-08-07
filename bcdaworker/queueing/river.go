/*
Package queueing implements "job" processing for bulk export requests

Job Processing is handled by RiverQueue and consists of three main components:
1. ProcessJob: Main job, ie bulk export requests.
2. PrepareJob: Handles logic dedicated to creating subjobs for ProcessJob.
3. CleanupJob: Handles cleaning up old/archived bulk export job files.

There are three workers for each step above; they are assigned a "kind" of work and do that work only.

When a request comes in, the PrepareWorker will divide the steps into multiple pieces to be worked,
depending on the number of beneficiaries and resources requested. Each of those pieces will enqueue a new Job which will be picked up by a jobProcessWorker.

Jobs are written to the application database. Jobs contain set of keys, which are generated in step 2 and then made available for the consumer that made the request.
*/
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
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/robfig/cron/v3"
	sloglogrus "github.com/samber/slog-logrus"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

type Notifier interface {
	PostMessageContext(context.Context, string, ...slack.MsgOption) (string, string, error)
}

// TODO: better dependency injection (db, worker, logger).  Waiting for pgxv5 upgrade
func StartRiver(db *sql.DB, numWorkers int) *queue {
	pool := database.ConnectPool()

	workers := river.NewWorkers()
	prepareWorker, err := NewPrepareJobWorker(db)
	if err != nil {
		panic(err)
	}
	river.AddWorker(workers, &JobWorker{db: db})
	river.AddWorker(workers, NewCleanupJobWorker(db))
	river.AddWorker(workers, prepareWorker)

	schedule, err := cron.ParseStandard("0 11,23 * * *")

	if err != nil {
		panic("Invalid cron schedule")
	}

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			schedule,
			func() (river.JobArgs, *river.InsertOpts) {
				return worker_types.CleanupJobArgs{}, &river.InsertOpts{}
			},
			&river.PeriodicJobOpts{},
		),
	}

	logger := getSlogLogger()

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: numWorkers},
		},
		// TODO: whats an appropriate timeout?
		JobTimeout:   -1, // default for river is 1m, using -1 for no timeout
		Logger:       logger,
		Workers:      workers,
		PeriodicJobs: periodicJobs,
	})

	if err != nil {
		logger.Error("failed to init river client", "error", err)
		panic(err)
	}

	if err := riverClient.Start(context.Background()); err != nil {
		logger.Error("failed to start river client", "error", err)
		panic(err)
	}

	q := &queue{
		ctx:        context.Background(),
		client:     riverClient,
		worker:     worker.NewWorker(db),
		repository: postgres.NewRepository(db),
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
