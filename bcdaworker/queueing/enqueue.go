/*
Enqueue.go has an interface and a method for instantiating a new River Client that satisfies the Enqueuer interface.
This allows the River client to be mocked for testing.
*/

package queueing

import (
	"context"
	"database/sql"

	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"

	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Enqueuer only handles inserting job entries into the appropriate table
type Enqueuer interface {
	AddJob(ctx context.Context, job worker_types.JobEnqueueArgs, priority int) error
	AddPrepareJob(ctx context.Context, job worker_types.PrepareJobArgs) error
}

// Creates a river client for the Job queue. If the client does not call .Start(), then it is insert only
// We still need the workers and the types of workers to insert them
func NewEnqueuer(db *sql.DB, pool *pgxv5Pool.Pool) Enqueuer {
	workers := river.NewWorkers()
	river.AddWorker(workers, &JobWorker{db: db})
	prepareWorker, err := NewPrepareJobWorker(db)
	if err != nil {
		panic(err)
	}
	river.AddWorker(workers, prepareWorker)

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		MaxAttempts: 6, // This is roughly 17m of total attempts with exp backoff
		Workers:     workers,
	})
	if err != nil {
		panic(err)
	}

	return riverEnqueuer{riverClient}
}

// RIVER implementation https://github.com/riverqueue/river
type riverEnqueuer struct {
	*river.Client[pgxv5.Tx]
}

func (q riverEnqueuer) AddJob(ctx context.Context, job worker_types.JobEnqueueArgs, priority int) error {
	// TODO: convert this to use transactions (q.InsertTx), likely only possible after upgrade to pgxv5
	// This could also be refactored to a batch insert: riverClient.InsertManyTx or riverClient.InsertMany
	_, err := q.Insert(ctx, job, &river.InsertOpts{
		Priority: priority,
	})
	if err != nil {
		return err
	}

	return err
}

func (q riverEnqueuer) AddPrepareJob(ctx context.Context, job worker_types.PrepareJobArgs) error {
	_, err := q.Insert(ctx, job, nil)
	if err != nil {
		return err
	}

	return err
}
