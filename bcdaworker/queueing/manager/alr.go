package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"

	// The follow two packages imported to use repository.ErrJobNotFound etc.
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/bgentry/que-go"
	"github.com/sirupsen/logrus"
)

/******************************************************************************
	Data Structures
******************************************************************************/

// alrQueue is the data structure for jobs related to Assignment List Report
// (ALR). ALR piggybacks Beneficiary FHIR through the masterQueue data struct.
// Ensure there is no field clashes with queue data struct.
type alrQueue struct {
	alrLog    *logrus.Logger
	alrWorker worker.AlrWorker
}

/******************************************************************************
	Functions
******************************************************************************/

// checkIFCanncelled was originally a closure to check if the job was cancelled,
// but it has been turned into a func for ALR for clarity
func checkIfCancelled(ctx context.Context, q *masterQueue,
	cancel context.CancelFunc, jobArgs models.JobAlrEnqueueArgs) {
	for {
		select {
		case <-time.After(15 * time.Second):
			jobStatus, err := q.repository.GetJobByID(ctx, jobArgs.ID)

			if err != nil {
				q.alrLog.Warnf("Could not find job %d status: %s", jobArgs.ID, err)
			}

			if jobStatus.Status == models.JobStatusCancelled {
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

/******************************************************************************
	Methods
******************************************************************************/

// startALRJob is the Job that the worker will run from the pool. This function
// has been written here (alr.go) to separate from beneficiary FHIR workflow.
// This job is handled by the same worker pool that works on beneficiary.
func (q *masterQueue) startAlrJob(job *que.Job) error {

	// Creating Context for possible cancellation; used by checkIfCancelled fn
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Unmarshall JSON that contains the job details
	var jobArgs models.JobAlrEnqueueArgs
	err := json.Unmarshal(job.Args, &jobArgs)
	// If we cannot unmarshall this, it would be very problematic.
	if err != nil {
		// TODO: perhaps just fail the job?
		q.alrLog.Warnf("Failed to unmarhall job.Args '%s' %s.",
			job.Args, err)
	}

	// Check if the job was cancelled
	go checkIfCancelled(ctx, q, cancel, jobArgs)

	// Validate the job like bcdaworker/worker/worker.go#L43
	// TODO: Abstract this into either a function or interface like the bfd worker
	alrJobs, err := q.repository.GetJobByID(ctx, jobArgs.ID)
	if err != nil { // if this is not nil, we could not find the Job
		// Drill down to what kind of error this is...
		if errors.Is(err, repository.ErrJobNotFound) {
			// Parent job is not found
			// If parent job is not found reach maxretry, fail the job
			if job.ErrorCount >= q.MaxRetry {
				q.alrLog.Errorf("No job found for ID: %d acoID: %s. Retries exhausted. Removing job from queue.", jobArgs.ID,
					jobArgs.CMSID)
				// By returning a nil error response, we're singaling to que-go to remove this job from the jobqueue.
				return nil
			}
			q.alrLog.Warnf("No Job with ID %d ACO %s found. Will retry.", jobArgs.ID, jobArgs.CMSID)
			return fmt.Errorf("could not retrieve job from database")
		}
		// Else that job just doesn't exist
		return fmt.Errorf("failed to valiate job: %w", err)
	}
	// If the job was cancalled...
	if alrJobs.Status == models.JobStatusCancelled {
		q.alrLog.Warnf("ALR big job has been cancelled, worker will not be tasked for %s",
			job.Args)
		return nil
	}
	// If the job has been failed by a previous worker...
	if alrJobs.Status == models.JobStatusFailed {
		q.alrLog.Warnf("ALR big job has been failed, worker will not be tasked for %s",
			job.Args)
		return nil
	}
	// End of validation

	// Before moving forward, check if this job has failed before
	// If it has reached the maxRetry, stop the parent job
	if job.ErrorCount > q.MaxRetry {
		// Fail the job - ALL OR NOTHING
		err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusFailed)
		if err != nil {
			q.alrLog.Warnf("Could not mark job %d as failed in DB.",
				jobArgs.ID)
		}
		q.alrLog.Warnf("One of the job for '%d' has failed five times.",
			jobArgs.ID)
		// Clean up the failed job from tracker
		return nil
	}

	// Check the status of the job, and put it in to progess if needed
	err = q.repository.UpdateJobStatusCheckStatus(ctx, jobArgs.ID, models.JobStatusPending,
		models.JobStatusInProgress)
	if err != nil {
		// This is a little confusing, but if the job is not updated b/c it's still in pending
		// don't fail the job.
		if !errors.Is(err, repository.ErrJobNotUpdated) {
			q.alrLog.Warnf("Failed to update job status '%s' %s.",
				job.Args, err)
			return err
		}
	}

	// Run ProcessAlrJob, which is the meat of the whole operation
	err = q.alrWorker.ProcessAlrJob(ctx, jobArgs)
	if err != nil {
		// This means the job did not finish
		q.alrLog.Warnf("Failed to complete job.Args '%s' %s", job.Args, err)
		// Re-enqueue the job
		return err
	}

	// Since the completed count is used for reporting (not for job completion), we do not
	// need it to succeed for moving on.
	if err := q.repository.IncrementCompletedJobCount(ctx, jobArgs.ID); err != nil {
		q.alrLog.Warnf("Failed to increment completed count %s", err.Error())
	}

	jobComplete, err := q.isJobComplete(ctx, jobArgs.ID)
	if err != nil {
		q.alrLog.Warnf("Failed to check job completion %s", err)
		return err
	}

	if jobComplete {
		// Finished writing all data, we can now move the data over to the payload directory
		err := os.Rename(fmt.Sprintf("%s/%d", q.StagingDir, jobArgs.ID), fmt.Sprintf("%s/%d", q.PayloadDir, jobArgs.ID))
		if err != nil {
			q.alrLog.Warnf("Failed to move data to payload directory %s", err)
			return err
		}

		err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusCompleted)
		if err != nil {
			// This means the job did not finish for various reason
			q.alrLog.Warnf("Failed to update job to complete for '%s' %s", job.Args, err)
			// Re-enqueue the job
			return err
		}
	}

	return nil
}

func (q *masterQueue) isJobComplete(ctx context.Context, jobID uint) (bool, error) {
	j, err := q.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return false, fmt.Errorf("failed to get job: %w", err)
	}

	switch j.Status {
	case models.JobStatusCompleted:
		return true, nil
	case models.JobStatusCancelled, models.JobStatusFailed:
		// Terminal status stop checking
		q.alrLog.Warnf("Failed to mark job as completed (Job %s): %s", j.Status, err)
		return false, nil
	}

	completedCount, err := q.repository.GetJobKeyCount(ctx, jobID)
	if err != nil {
		return false, fmt.Errorf("failed to get job key count: %w", err)
	}

	if completedCount > j.JobCount {
		q.alrLog.WithFields(logrus.Fields{
			"jobID":    j.ID,
			"jobCount": j.JobCount, "completedJobCount": completedCount}).
			Warn("Excess number of jobs completed.")
	}
	return completedCount >= j.JobCount, nil
}
