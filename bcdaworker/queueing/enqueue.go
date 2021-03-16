package queueing

import (
	"encoding/json"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/bgentry/que-go"
	log "github.com/sirupsen/logrus"
)

const (
	QUE_PROCESS_JOB = "ProcessJob"
)

type Enqueuer interface {
	AddJob(job models.JobEnqueueArgs, priority int) error
}

func NewEnqueuer() Enqueuer {
	db := database.QueueConnection
	conn, err := db.Acquire()
	if err != nil {
		log.Fatalf("Failed to get queue connection %s", err.Error())
	}
	defer db.Release(conn)

	if err := que.PrepareStatements(conn); err != nil {
		log.Fatalf("Failed to setup prepared statements %s", err.Error())
	}

	return queEnqueuer{que.NewClient(db)}
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
		Type:     QUE_PROCESS_JOB,
		Args:     args,
		Priority: int16(priority),
	}

	return q.Enqueue(j)
}
