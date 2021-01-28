package queueing

import (
	"encoding/json"
	"log"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/manager"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
)

type Enqueuer interface {
	AddJob(job models.JobEnqueueArgs, priority int) error
}

func NewEnqueuer(queueDatabaseURL string) Enqueuer {
	cfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	pool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   cfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		log.Fatal(err)
	}

	return queEnqueuer{que.NewClient(pool)}
}

type queEnqueuer struct {
	*que.Client
}

func (q queEnqueuer) AddJob(job models.JobEnqueueArgs, priority int) error {
	args, err := json.Marshal(job)
	if err != nil {
		return err
	}

	j := &que.Job{
		Type:     manager.QUE_PROCESS_JOB,
		Args:     args,
		Priority: int16(priority),
	}

	return q.Enqueue(j)
}
