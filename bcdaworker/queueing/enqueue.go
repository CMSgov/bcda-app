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

const (
	QUE_PROCESS_JOB = "ProcessJob"
	ALR_JOB         = "AlrJob"
)

type Enqueuer interface {
	AddJob(ctx context.Context, job models.JobEnqueueArgs, priority int) error
	AddAlrJob(ctx context.Context, job models.JobAlrEnqueueArgs, priority int) error
}

func NewEnqueuer() Enqueuer {
	if conf.GetEnv("QUEUE_LIBRARY") == "river" {
		riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Connection), &river.Config{
			Workers: river.NewWorkers(),
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
		Type:     QUE_PROCESS_JOB,
		Args:     args,
		Priority: int16(p),
	}

	return q.Enqueue(j)
}

// ALR ENQ...
func (q queEnqueuer) AddAlrJob(ctx context.Context, job models.JobAlrEnqueueArgs, priority int) error {
	args, err := json.Marshal(job)
	if err != nil {
		return err
	}

	p, e := safecast.ToInt16(priority)
	if e != nil {
		return e
	}

	j := &que.Job{
		Type:     ALR_JOB,
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
	_, err := q.Insert(context.Background(), job, nil)
	return err
}

func (q riverEnqueuer) AddAlrJob(ctx context.Context, job models.JobAlrEnqueueArgs, priority int) error {
	return nil
}
