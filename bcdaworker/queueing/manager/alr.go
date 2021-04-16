package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"

	// The follow two packages imported to use repository.ErrJobNotFound
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"
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

// Max number of retries either set by ENV or default value of 3
var maxRetry = int32(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))

// To reduce the number of pings to the DB, internal tracking of jobs
// Pre-allocation of 512 is abitraru... seems like a good avg number
var alrJobTracker = make(map[uint]struct{}, 512)

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

// deleteAndRetry is a helper function that will delete previous attempt at
// creating ndjson if the worker hit an error along the way. This is to avoid
// serving incomplete or redundant data to user.
func deleteAndRetry(l *logrus.Logger, dir string, id uint, file string) {
	err := os.Remove(fmt.Sprintf("%s/%d/%s.ndjson", dir, id, file))
	if err != nil {
		l.Fatal("Could not remove alr ndjson.")
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

	// Validate the job like bcdaworker/worker/worker.go#L43
	// TODO: Abstract this into either function or interface like the bfd worker
	alrJobs, err := q.repository.GetJobByID(ctx, jobArgs.ID)
	if err != nil { // if this is not nil, we could not find the Job
		// Drill down to what kind of error this is...
		if errors.Is(err, repository.ErrJobNotFound) {
			// Parent job is not found
			if job.ErrorCount >= maxRetry {
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
	if alrJobs.Status == models.JobStatusCancelled {
		q.alrLog.Warnf("ALR big job has been cancelled, worker will not tasked for %s",
			job.Args)
		return nil
	}

	// Keep track of how many times a small job has failed.
	// If the threshold is reached, fail the job
	if _, exists := alrJobTracker[alrJobs.ID]; !exists {

		// So we have a brand new job... we will start tracking it
		alrJobTracker[alrJobs.ID] = struct{}{}

		err := q.repository.UpdateJobStatus(ctx, jobArgs.ID,
			models.JobStatusInProgress)
		if err != nil {
			q.alrLog.Warnf("Failed to update job status '%s' %s.",
				job.Args, err)
			// unlock before returning to avoid deadlock
			return err
		}
	} else {
		// Check if this job has been retried too many times - 5 times is max
		if job.ErrorCount > maxRetry {
			// Fail the job
			err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusFailed)
			if err != nil {
				q.alrLog.Warnf("Could not mark job %d as failed in DB.",
					jobArgs.ID)
			}
			q.alrLog.Warnf("One of the job for '%d' has failed five times.",
				jobArgs.ID)
			// Clean up the failed job from tracker
			delete(alrJobTracker, jobArgs.ID)
			return nil
		}

	}

	// Check if the job was cancelled
	go checkIfCancelled(ctx, q, cancel, jobArgs)

	// Run ProcessAlrJob, which is the meat of the whole operation
	ndjsonFilename := uuid.NewRandom()
	err = q.alrWorker.ProcessAlrJob(ctx, jobArgs, ndjsonFilename)
	if err != nil {
		// This means the job did not finish
		q.alrLog.Warnf("Failed to complete job.Args '%s' %s", job.Args, err)
		// Re-enqueue the job
		return err
	}

	// Update DB that work is done / success
	err = q.repository.IncrementCompletedJobCount(ctx, jobArgs.ID)
	if err != nil {
		q.alrLog.Warnf("Failed to increment job count for '%s' %s", job.Args, err)
		// Can't increment for some DB reason... rollback the file created and try again...
		deleteAndRetry(q.alrLog, q.alrWorker.FHIR_STAGING_DIR, jobArgs.ID,
			string(ndjsonFilename))
		return err
	}

	alrJobs, err = q.repository.GetJobByID(ctx, jobArgs.ID)
	if err != nil {
		q.alrLog.Warnf("Failed to get alr Job by id for '%s' %s", job.Args, err)
		// Try again
		deleteAndRetry(q.alrLog, q.alrWorker.FHIR_STAGING_DIR, jobArgs.ID,
			string(ndjsonFilename))
		return err
	}

	// Check if the Job is done
	if alrJobs.CompletedJobCount == alrJobs.JobCount {
		// we're done, so move files from staging to payload
		// First make sure there is no conflicting directory... only an issue on local
		err := os.RemoveAll(fmt.Sprintf("%s/%d", q.alrWorker.FHIR_PAYLOAD_DIR, jobArgs.ID))
		if err != nil {
			q.alrLog.Warnf("Could not clear payload directory")
		}

		// Move the file from staging to payload
		err = os.Rename(fmt.Sprintf("%s/%d", q.alrWorker.FHIR_STAGING_DIR, jobArgs.ID),
			fmt.Sprintf("%s/%d", q.alrWorker.FHIR_PAYLOAD_DIR, jobArgs.ID))
		if err != nil {
			q.alrLog.Warnf("Failed to rename alr dirs '%s' %s", job.Args, err)
			deleteAndRetry(q.alrLog, q.alrWorker.FHIR_STAGING_DIR, jobArgs.ID,
				string(ndjsonFilename))
			return err
		}
		// mark job as done
		err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusCompleted)

		if err != nil {
			q.alrLog.Warnf("Failed to update job to complete '%s' %s", job.Args, err)
			deleteAndRetry(q.alrLog, q.alrWorker.FHIR_STAGING_DIR, jobArgs.ID,
				string(ndjsonFilename))
			return err
		}

		// Everything went smoothly, let's clean up the tracker
		delete(alrJobTracker, jobArgs.ID)
	}

	return nil
}
