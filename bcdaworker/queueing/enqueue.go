package queueing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/manager"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
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

	// Ensure that the connections are valid. Needed until we move to pgx v4
	database.StartHealthCheck(context.Background(), pool, 10*time.Second)

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
