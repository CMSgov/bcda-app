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
	"log/slog"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/robfig/cron/v3"
	"github.com/slack-go/slack"
)

type Notifier interface {
	PostMessageContext(context.Context, string, ...slack.MsgOption) (string, string, error)
}

func CreateRiverClient(logger *slog.Logger, db *sql.DB, numWorkers int) *river.Client[pgx.Tx] {
	pool := database.ConnectPool()
	workers := river.NewWorkers()
	prepareWorker, err := NewPrepareJobWorker(db, pool)
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

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: numWorkers},
		},
		JobTimeout:      10 * time.Minute,
		Logger:          logger,
		MaxAttempts:     8, // This is roughly an hour of retries
		Workers:         workers,
		PeriodicJobs:    periodicJobs,
		SoftStopTimeout: 30 * time.Second,
	})

	if err != nil {
		logger.Error("failed to init river client", "error", err)
		panic(err)
	}

	return riverClient
}
