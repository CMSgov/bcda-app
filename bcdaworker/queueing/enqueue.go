/*
Enqueue.go has an interface and a method for instantiating a new River Client that satisfies the Enqueuer interface.
This allows the River client to be mocked for testing.
*/

package queueing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/bgentry/que-go"
	"github.com/ccoveille/go-safecast"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Enqueuer only handles inserting job entries into the appropriate table
type Enqueuer interface {
	AddJob(ctx context.Context, job models.JobEnqueueArgs, priority int) error
	AddPrepareJob(ctx context.Context, job PrepareJobArgs) error
	AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error
}

/* Creates a river client for the PrepareJob queue. If the client does not call .Start(), then it is insert only.*/
func NewPrepareEnqueuer() Enqueuer {
	if conf.GetEnv("QUEUE_LIBRARY") == "river" {
		workers := river.NewWorkers()
		prepareWorker, err := NewPrepareJobWorker()
		if err != nil {
			fmt.Printf("failed at newprepareworker()")
			panic(err)
		}
		river.AddWorker(workers, prepareWorker)

		riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Pool), &river.Config{
			Workers: workers,
		})
		if err != nil {
			panic(err)
		}

		return riverEnqueuer{riverClient}
	}

	return queEnqueuer{que.NewClient(database.QueueConnection)}
}

/* Creates a river client for the Job queue. If the client does not call .Start(), then it is insert only.*/
func NewEnqueuer() Enqueuer {
	if conf.GetEnv("QUEUE_LIBRARY") == "river" {
		workers := river.NewWorkers()
		river.AddWorker(workers, &JobWorker{})

		riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Pool), &river.Config{
			Workers: workers,
		})
		if err != nil {
			panic(err)
		}

		return riverEnqueuer{riverClient}
	}

	return queEnqueuer{que.NewClient(database.QueueConnection)}
}

// Deprecated: User River Queue instead.
type queEnqueuer struct {
	*que.Client
}

// Deprecated: User River Queue instead.
func (q queEnqueuer) AddJob(ctx context.Context, job models.JobEnqueueArgs, priority int) error {
	args, err := json.Marshal(job)
	if err != nil {
		return err
	}

	p, e := safecast.ToInt16(priority)
	if e != nil {
		return e
	}

	j := &que.Job{
		Type:     models.QUE_PROCESS_JOB,
		Args:     args,
		Priority: int16(p),
	}

	return q.Enqueue(j)
}

// Deprecated: User River Queue instead.
func (q queEnqueuer) AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error {
	args, err := json.Marshal(job)
	if err != nil {
		return err
	}

	p, e := safecast.ToInt16(priority)
	if e != nil {
		return e
	}

	j := &que.Job{
		Type:     models.ALR_JOB,
		Args:     args,
		Priority: int16(p),
	}

	return q.Enqueue(j)
}

// Deprecated: User River Queue instead.
func (q queEnqueuer) AddPrepareJob(ctx context.Context, job PrepareJobArgs) error {
	return nil
}

// RIVER implementation https://github.com/riverqueue/river
type riverEnqueuer struct {
	*river.Client[pgxv5.Tx]
}

func (q riverEnqueuer) AddJob(ctx context.Context, job models.JobEnqueueArgs, priority int) error {
	// TODO: convert this to use transactions (q.InsertTx), likely only possible after removal of que-go AND upgrade to pgxv5
	// This could also be refactored to a batch insert: riverClient.InsertManyTx or riverClient.InsertMany
	_, err := q.Insert(ctx, job, &river.InsertOpts{
		Priority: priority,
	})
	if err != nil {
		return err
	}

	return err
}

func (q riverEnqueuer) AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error {
	return nil
}

func (q riverEnqueuer) AddPrepareJob(ctx context.Context, job PrepareJobArgs) error {
	_, err := q.Insert(ctx, job, nil)
	if err != nil {
		return err
	}

	return err
}
