package queueing

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProcessJobFailedValidation_Integration(t *testing.T) {
	tests := []struct {
		name        string
		validateErr error
		expectedErr error
		expLogMsg   string
	}{
		{"ParentJobCancelled", worker.ErrParentJobCancelled, nil, `QJob \d+ associated with a cancelled parent Job \d+. Removing job from queue.`},
		{"ParentJobFailed", worker.ErrParentJobFailed, nil, `QJob \d+ associated with a failed parent Job \d+. Removing job from queue.`},
		{"NoBasePath", worker.ErrNoBasePathSet, nil, `QJob \d+ does not contain valid base path. Removing job from queue.`},
		{"NoParentJob", worker.ErrParentJobNotFound, repository.ErrJobNotFound, `No job found for Job: \d+ acoID: .+. Will retry.`},
		{"NoParentJobRetriesExceeded", worker.ErrParentJobNotFound, nil, `No job found for Job: \d+ acoID: .+. Retries exhausted. Removing job from queue.`},
		{"QueJobAlreadyProcessed", worker.ErrQueJobProcessed, nil, `QJob \d+ already processed for parent Job: \d+. Checking completion status and removing job from queue.`},
		{"OtherError", fmt.Errorf(constants.DefaultError), fmt.Errorf(constants.DefaultError), ""},
	}
	hook := test.NewLocal(testUtils.GetLogger(log.Worker))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := &worker.MockWorker{}
			defer worker.AssertExpectations(t)

			repo := repository.NewMockRepository(t)

			id, err := safecast.ToUint(1)
			if err != nil {
				t.Fatal(err)
			}
			job := models.Job{ID: id}

			jobid, e := safecast.ToInt(1)
			if e != nil {
				t.Fatal(e)
			}
			jobArgs := worker_types.JobEnqueueArgs{ID: jobid, ACOID: uuid.New()}
			qJobID := int64(1)

			if tt.name == "QueJobAlreadyProcessed" {
				job.Status = models.JobStatusCompleted
				repo.On("GetJobByID", testUtils.CtxMatcher, job.ID).Return(&job, nil)
				// Note: GetJobKeyCount is not called when job status is already Completed
			}

			jobID, err := safecast.ToInt64(job.ID)
			if err != nil {
				t.Fatal(err)
			}

			worker.On("ValidateJob", testUtils.CtxMatcher, qJobID, jobArgs).Return(nil, tt.validateErr)

			logger := testUtils.GetLogger(log.Worker)
			if logger == nil {
				t.Fatal("Logger is nil")
			}

			var exportJob *models.Job
			var ackJob bool

			errorCount := 0
			if tt.name == "NoParentJobRetriesExceeded" {
				errorCount = 10
			}

			config := ValidateJobConfig{
				WorkerInstance: worker,
				Logger:         logger,
				Repository:     repo,
				JobID:          jobID,
				QJobID:         qJobID,
				Args:           jobArgs,
				ErrorCount:     errorCount,
			}

			exportJob, err, ackJob = validateJob(context.Background(), config)

			if tt.expectedErr == nil {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q but got no error", tt.expectedErr.Error())
				} else {
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				}
			}

			if tt.expLogMsg != "" {
				if hook.LastEntry() != nil {
					assert.Regexp(t, regexp.MustCompile(tt.expLogMsg), hook.LastEntry().Message)
				} else {
					t.Errorf("Expected log message but no log entry found")
				}
			}

			if ackJob {
				assert.Nil(t, exportJob)
			}
		})
	}
}

func TestCheckIfCancelled(t *testing.T) {
	q := queue{
		repository: nil,
	}

	mockRepo := repository.MockRepository{}
	mockRepo.On("GetJobByID", testUtils.CtxMatcher, mock.Anything).Return(
		&models.Job{
			Status: models.JobStatusInProgress,
		},
		nil,
	).Once()
	mockRepo.On("GetJobByID", testUtils.CtxMatcher, mock.Anything).Return(
		&models.Job{
			Status: models.JobStatusCancelled,
		},
		nil,
	)
	q.repository = &mockRepo

	ctx, cancel := context.WithCancel(context.Background())
	jobs := worker_types.JobEnqueueArgs{}

	jobID, err := safecast.ToInt64(jobs.ID)
	assert.NoError(t, err)

	// In production we wait 15 second intervals, for test we do 1
	go checkIfCancelled(ctx, q.repository, cancel, jobID, 1)

	// Check if the context has been cancelled
	var cnt uint8
	var success bool

Outer:
	for {
		select {
		case <-time.After(time.Second):
			if cnt > 10 {
				break Outer
			}
			cnt++
		case <-ctx.Done():
			success = true
			break Outer
		}
	}

	assert.True(t, success)
}
