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
	AddJob(ctx context.Context, tx pgxv5.Tx, job worker_types.JobEnqueueArgs, priority int) error
	AddPrepareJob(ctx context.Context, job worker_types.PrepareJobArgs) error
}

// Creates a river client for the Job queue. If the client does not call .Start(), then it is insert only
// We still need the workers and the types of workers to insert them
func NewEnqueuer(db *sql.DB, pool *pgxv5Pool.Pool) Enqueuer {
	workers := river.NewWorkers()
	river.AddWorker(workers, &JobWorker{db: db})
	prepareWorker, err := NewPrepareJobWorker(db, pool)
	if err != nil {
		panic(err)
	}
	river.AddWorker(workers, prepareWorker)

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		MaxAttempts: 8, // This is a few hours worth of retries
		Workers:     workers,
	})
	if err != nil {
		panic(err)
	}

	return riverEnqueuer{pool: pool, Client: riverClient}
}

// RIVER implementation https://github.com/riverqueue/river
type riverEnqueuer struct {
	pool *pgxv5Pool.Pool

	*river.Client[pgxv5.Tx]
}

func (q riverEnqueuer) AddJob(ctx context.Context, tx pgxv5.Tx, job worker_types.JobEnqueueArgs, priority int) error {
	_, err := q.InsertTx(ctx, tx, job, &river.InsertOpts{
		Priority: priority,
	})
	if err != nil {
		return err
	}

	return nil
}

func (q riverEnqueuer) AddPrepareJob(ctx context.Context, job worker_types.PrepareJobArgs) error {
	_, err := q.Insert(ctx, job, nil)
	if err != nil {
		return err
	}

	return err
}
