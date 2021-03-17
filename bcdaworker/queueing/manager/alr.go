package manager

import (
	"context"
	"encoding/json"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/bgentry/que-go"
	"github.com/sirupsen/logrus"
)

// alrQueue is the data structure for jobs related to Assignment List Report
// (ALR). ALR piggybacks Beneficiary FHIR through the masterQueue data struct.
// Ensure there is no field clashes with queue data struct.
type alrQueue struct {
	alrLog    *logrus.Logger
	alrWorker worker.AlrWorker
}

// startALRJob is the Job that the worker will run from the pool. This function
// has been written here (alr.go) to separate from beneficiary FHIR workflow.
// This job is handled by the same worker pool that works on beneficiary.
func (q *masterQueue) startAlrJob(job *que.Job) error {
	// Creating Context for possible cancellation
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Unmarchall JSON that contains the job details
	// JSON selected for ALR for continuity... may change in the future
	var jobArgs models.JobAlrEnqueueArgs
	err := json.Unmarshal(job.Args, &jobArgs)
	if err != nil {
		q.alrLog.Warnf("Failed to unmarhall job.Args '%s' %s. Removing job...",
			job.Args, err)
		// By returning nil to que-go, job is removed from queue
		return nil
	}

	// Check if the job was cancelled
	go func() {
		for {
			select {
			case <-time.After(15 * time.Second):
				jobStatus, err := q.repository.GetJobByID(ctx, jobArgs.ID)

				if err != nil {
					q.alrLog.Warnf("Could not find job %d status: %s", jobArgs.ID, err)
				}

				if jobStatus.Status == models.JobStatusCancelled {
					// cancelled context will get picked up by worker.go#writeBBDataToFile
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Do the Job
	err = q.alrWorker.ProcessAlrJob(ctx, jobArgs)
	if err != nil {
		// This means the job did not finish for various reason
		q.alrLog.Warnf("Failed to complete job.Args '%s' %s", job.Args, err)
		// Re-enqueue the job
		return err
	}

	// Update DB that work is done / success
	err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusCompleted)
	if err != nil {
		// This means the job did not finish for various reason
		q.alrLog.Warnf("Failed to update job to complete for '%s' %s", job.Args, err)
		// Re-enqueue the job
		return nil
	}

	return nil
}
