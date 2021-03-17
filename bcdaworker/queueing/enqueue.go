package queueing

import (
	"encoding/json"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/bgentry/que-go"
)

const (
	QUE_PROCESS_JOB = "ProcessJob"
	ALR_JOB         = "AlrJob"
)

type Enqueuer interface {
	AddJob(job models.JobEnqueueArgs, priority int) error
	AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error
}

func NewEnqueuer() Enqueuer {
	return queEnqueuer{que.NewClient(database.QueueConnection)}
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

// ALR ENQ...
func (q queEnqueuer) AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error {
	args, err := json.Marshal(job)
	if err != nil {
		return err
	}

	j := &que.Job{
		Type:     ALR_JOB,
		Args:     args,
		Priority: int16(priority),
	}

	return q.Enqueue(j)
}
