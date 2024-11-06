package manager

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/sirupsen/logrus"
)

func StartRiver(log logrus.FieldLogger, numWorkers int) *MasterQueue {
	riverClient, err := river.NewClient(riverpgxv5.New(database.Pgxv5Connection), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: numWorkers},
		},
		Workers: river.NewWorkers(),
	})
	if err != nil {
		panic(err)
	}

	if err := riverClient.Start(context.Background()); err != nil {
		panic(err)
	}

	return &MasterQueue{}
}

func (q queue) StopRiver() {
	if err := q.client.Stop(q.ctx); err != nil {
		panic(err)
	}
}

// type JobWorker struct {
// 	// An embedded WorkerDefaults sets up default methods to fulfill the rest of
// 	// the Worker interface:
// 	river.WorkerDefaults[models.JobEnqueueArgs]
// }

// func (w *JobWorker) Work(ctx context.Context, job *river.Job[models.JobEnqueueArgs]) error {
// 	// sort.Strings(job.Args.Strings)
// 	// fmt.Printf("Sorted strings: %+v\n", job.Args.Strings)
// 	// return nil
// }
