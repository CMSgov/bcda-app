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

type alrJobTrackerStuct struct {
	tracker map[uint]models.JobStatus
	sync.Mutex
}

var alrJobTracker = alrJobTrackerStuct{
	tracker: make(map[uint]models.JobStatus, 2),
}

/******************************************************************************
	Functions
	- checkIfCancelled - originally a closure, turned into func for clarity
******************************************************************************/

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

	// Unmarchall JSON that contains the job details
	// JSON selected for ALR for continuity... may change in the future
	var jobArgs models.JobAlrEnqueueArgs
	err := json.Unmarshal(job.Args, &jobArgs)
	if err != nil {
		q.alrLog.Warnf("Failed to unmarhall job.Args '%s' %s.",
			job.Args, err)
		// By returning nil to que-go, job is removed from queue
		return err
	}

	// To reduce the number of pings to the DB, track some information on the
	// worker side.
	// This is actually not a race condition, but the unit-test does not like it.
	// alrJobTracker should only be updated twice: first and last job.
	alrJobTracker.Lock()
	if _, exists := alrJobTracker.tracker[jobArgs.ID]; !exists {
		alrJobTracker.tracker[jobArgs.ID] = models.JobStatusInProgress
		err := q.repository.UpdateJobStatus(ctx, jobArgs.ID,
			models.JobStatusInProgress)
		if err != nil {
			q.alrLog.Warnf("Failed to update job status '%s' %s.",
				job.Args, err)
			// unlock before returning to prevent deadlock
			alrJobTracker.Unlock()
			// By returning nil to que-go, job is removed from queue
			return err
		}
	}
	alrJobTracker.Unlock()

	// Check if the job was cancelled
	go checkIfCancelled(ctx, q, cancel, jobArgs)

	// Do the Job
	err = q.alrWorker.ProcessAlrJob(ctx, jobArgs)
	if err != nil {
		// This means the job did not finish for various reason
		q.alrLog.Warnf("Failed to complete job.Args '%s' %s", job.Args, err)
		// Re-enqueue the job
		return err
	}

	// Update DB that work is done / success
	err = q.repository.IncrementCompletedJobCount(ctx, jobArgs.ID)
	if err != nil {
		q.alrLog.Warnf("Failed to increment job to count for '%s' %s", job.Args, err)
		// Can't increment for some DB reason... rollback the file created and try again...
		return err
	}

	alrJobs, err := q.repository.GetJobByID(ctx, jobArgs.ID)
	if err != nil {
		q.alrLog.Warnf("Failed to get alr Job by id for '%s' %s", job.Args, err)
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

			return err
		}
		// mark job as done
		err = q.repository.UpdateJobStatus(ctx, jobArgs.ID, models.JobStatusCompleted)

		if err != nil {
			q.alrLog.Warnf("Failed to update job to complete '%s' %s", job.Args, err)
			return err
		}
		alrJobTracker.Lock()
		delete(alrJobTracker.tracker, jobArgs.ID)
		alrJobTracker.Unlock()
	}

	return nil
}
