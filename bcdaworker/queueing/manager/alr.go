package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
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

// alrJobTrackerStruct is used to track the status of the job
type alrJobTrackerStuct struct {
	tracker map[uint]alrJobAttempt // unint here is the big job
	sync.Mutex
}

type alrJobAttempt struct {
	status         models.JobStatus
	attemptTracker map[uint]uint // 1st uint is the small job
}

// Instance at the global private level
var alrJobTracker = alrJobTrackerStuct{
	tracker: make(map[uint]alrJobAttempt),
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
		q.alrLog.Fatalf("Failed to unmarhall job.Args '%s' %s.",
			job.Args, err)
	}

	// Check if this is already a failed job, and don't continue if it is
	alrJobs, err := q.repository.GetJobByID(ctx, jobArgs.ID)
	if err != nil {
		q.alrLog.Warnf("Could not get information on '%s' %s.",
			job.Args, err)
		return nil
	}
	if alrJobs.Status == models.JobStatusCancelled {
		q.alrLog.Warnf("ALR big job has been cancelled, worker will not tasked for %s",
			job.Args)
		return nil
	}

	// Keep track of how many times a small job has failed.
	// If the threshold is reached, fail the job
	alrJobTracker.Lock()
	if _, exists := alrJobTracker.tracker[jobArgs.ID]; !exists {

		alrJobTracker.tracker[jobArgs.ID] = alrJobAttempt{
			status:         models.JobStatusInProgress,
			attemptTracker: make(map[uint]uint),
		}
		alrJobTracker.tracker[jobArgs.ID].attemptTracker[jobArgs.QueueID] = 0
		err := q.repository.UpdateJobStatus(ctx, jobArgs.ID,
			models.JobStatusInProgress)
		if err != nil {
			q.alrLog.Warnf("Failed to update job status '%s' %s.",
				job.Args, err)
			// unlock before returning to avoid deadlock
			alrJobTracker.Unlock()
			return err
		}
	} else {
		// Check if this job has been retried too many times - 5 times is max
		if alrJobTracker.tracker[jobArgs.ID].attemptTracker[jobArgs.QueueID] > 4 {
			// Fail the job
			err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusFailed)
			if err != nil {
				q.alrLog.Warnf("Could not mark job %d as failed in DB.",
					jobArgs.ID)
			}
			q.alrLog.Warnf("One of the job for '%d' has failed five times.",
				jobArgs.ID)
			// Clean up the failed job from tracker
			delete(alrJobTracker.tracker, jobArgs.ID)
			return nil
		}

		alrJobTracker.tracker[jobArgs.ID].attemptTracker[jobArgs.QueueID]++
	}
	alrJobTracker.Unlock()

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
		alrJobTracker.Lock()
		delete(alrJobTracker.tracker, jobArgs.ID)
		alrJobTracker.Unlock()
	}

	return nil
}
