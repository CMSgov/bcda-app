package queueing

import (
	"context"
	"encoding/json"

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
	AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error
}

func NewEnqueuer() Enqueuer {
	if conf.GetEnv("QUEUE_LIBRARY") == "river" {
		workers := river.NewWorkers()
		river.AddWorker(workers, &JobWorker{})

		riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Connection), &river.Config{
			Workers: workers,
		})
		if err != nil {
			panic(err)
		}

		return riverEnqueuer{riverClient}
	}

	return queEnqueuer{que.NewClient(database.QueueConnection)}
}

// QUE-GO implementation https://github.com/bgentry/que-go
type queEnqueuer struct {
	*que.Client
}

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

// ALR ENQ...
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

// RIVER implementation https://github.com/riverqueue/river
type riverEnqueuer struct {
	*river.Client[pgxv5.Tx]
}

func (q riverEnqueuer) AddJob(ctx context.Context, job models.JobEnqueueArgs, priority int) error {
	// TODO: convert this to use transactions (q.InsertTx), likely only possible after removal of que-go AND upgrade to pgxv5
	// This could also be refactored to a batch insert: riverClient.InsertManyTx or riverClient.InsertMany
	// opts := river.InsertOpts{Priority: priority}
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
