package queueing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/bgentry/que-go"
	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Assignment List Report (ALR) shares the worker pool and "piggy-backs" off
// Beneficiary FHIR Data workflow. Instead of creating redundant functions and
// methods, masterQueue wraps both structs allows for sharing.
type MasterQueue struct {
	*queue
	*alrQueue // This is defined in alr.go

	StagingDir string `conf:"FHIR_STAGING_DIR"`
	PayloadDir string `conf:"FHIR_PAYLOAD_DIR"`
	MaxRetry   int32  `conf:"BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES" conf_default:"3"`
}

func newMasterQueue(q *queue, qAlr *alrQueue) *MasterQueue {
	mq := &MasterQueue{
		queue:    q,
		alrQueue: qAlr,
	}

	if err := conf.Checkout(mq); err != nil {
		logrus.Fatal("Could not get data from conf for ALR.", err)
	}

	return mq
}

// StartQue creates a que-go client and begins listening for items
// It returns immediately since all of the associated workers are started
// in separate goroutines.
func StartQue(log logrus.FieldLogger, numWorkers int) *MasterQueue {
	// Allocate the queue in advance to supply the correct
	// in the workmap
	mainDB := database.Connection
	q := &queue{
		worker:        worker.NewWorker(mainDB),
		repository:    postgres.NewRepository(mainDB),
		log:           log,
		queDB:         database.QueueConnection,
		cloudWatchEnv: conf.GetEnv("DEPLOYMENT_TARGET"),
	}
	// Same as above, but do one for ALR
	qAlr := &alrQueue{
		alrLog:    log,
		alrWorker: worker.NewAlrWorker(mainDB),
	}
	master := newMasterQueue(q, qAlr)

	qc := que.NewClient(q.queDB)
	wm := que.WorkMap{
		models.QUE_PROCESS_JOB: q.processJob,
		models.ALR_JOB:         master.startAlrJob, // ALR currently shares pool
	}

	q.quePool = que.NewWorkerPool(qc, wm, numWorkers)

	q.quePool.Start()

	fmt.Printf("\n---START QUEUE: %+v\n", q)

	return master
}

// StopQue cleans up any resources created
func (q *MasterQueue) StopQue() {
	q.queDB.Close()
	q.quePool.Shutdown()
}

func (q *queue) processJob(queJob *que.Job) error {
	fmt.Printf("---Is que processjobbing: %+v\n", queJob)
	ctx, cancel := context.WithCancel(context.Background())

	defer q.updateJobQueueCountCloudwatchMetric()
	defer cancel()

	var jobArgs models.JobEnqueueArgs
	err := json.Unmarshal(queJob.Args, &jobArgs)
	if err != nil {
		// ACK the job because retrying it won't help us be able to deserialize the data
		q.log.Warnf("Failed to deserialize job.Args '%s' %s. Removing queuejob from que.", queJob.Args, err)
		return nil
	}

	fmt.Printf("\n---process job jobArgs: %+v", jobArgs)

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	ctx, _ = log.SetCtxLogger(ctx, "job_id", queJob.ID)
	ctx, logger := log.SetCtxLogger(ctx, "transaction_id", jobArgs.TransactionID)

	jobID, err := safecast.ToInt64(jobArgs.ID)
	if err != nil {
		return err
	}

	exportJob, err, ackJob := validateJob(ctx, ValidateJobConfig{
		WorkerInstance: q.worker,
		Logger:         logger,
		Repository:     q.repository,
		JobID:          jobID,
		QJobID:         queJob.ID,
		Args:           jobArgs,
		ErrorCount:     int(queJob.ErrorCount),
	})
	fmt.Printf("\n---exportJob: %+v\n", exportJob)
	fmt.Printf("---exportJob error: %+v\n", err)
	fmt.Printf("---exportJob ackJob: %+v\n", ackJob)
	if ackJob {
		// End logic here, basically acknowledge and return which will remove it from the queue.
		return nil
	}
	// Return error when we want to mark a job as having errored out, which will mark it to be retried
	if err != nil {
		return err
	}

	// start a goroutine that will periodically check the status of the parent job
	go checkIfCancelled(ctx, q.repository, cancel, queJob.ID, 15)

	fmt.Printf("---after check, %+v, %+v, %+v", queJob.ID, *exportJob, jobArgs)

	if err := q.worker.ProcessJob(ctx, queJob.ID, *exportJob, jobArgs); err != nil {
		err := errors.Wrap(err, "failed to process job")
		logger.Error(err)
		return err
	}

	return nil
}

func (q *queue) updateJobQueueCountCloudwatchMetric() {

	// Update the Cloudwatch Metric for job queue count
	if q.cloudWatchEnv != "" {
		sampler, err := metrics.NewSampler("BCDA", "Count")
		if err != nil {
			fmt.Println("Warning: failed to create new metric sampler...")
		} else {
			err := sampler.PutSample("JobQueueCount", q.getQueueJobCount(), []metrics.Dimension{
				{Name: "Environment", Value: q.cloudWatchEnv},
			})
			if err != nil {
				q.log.Error(err)
			}
		}
	}
}

func (q *queue) getQueueJobCount() float64 {
	row := q.queDB.QueryRow(`select count(*) from que_jobs;`)

	var count int
	if err := row.Scan(&count); err != nil {
		q.log.Error(err)
	}

	return float64(count)
}
