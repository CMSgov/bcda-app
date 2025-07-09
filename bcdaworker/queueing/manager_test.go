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
	"github.com/riverqueue/river"
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

			// Create a River job instead of que-go job
			riverJob := &river.Job[worker_types.JobEnqueueArgs]{
				Args: jobArgs,
			}

			// Set the error count to max to ensure that we've exceeded the retries
			if tt.name == "NoParentJobRetriesExceeded" {
				riverJob.Attempt = int(testUtils.CryptoRandInt31())
			}

			worker.On("ValidateJob", testUtils.CtxMatcher, int64(1), jobArgs).Return(nil, tt.validateErr)

			if tt.name == "QueJobAlreadyProcessed" {
				job.Status = models.JobStatusCompleted
				repo.On("GetJobByID", testUtils.CtxMatcher, job.ID).Return(&job, nil)
			}

			jobID, err := safecast.ToInt64(job.ID)
			if err != nil {
				t.Fatal(err)
			}

			exportJob, err, ackJob := validateJob(context.Background(), ValidateJobConfig{
				WorkerInstance: worker,
				Logger:         testUtils.GetLogger(log.Worker),
				Repository:     repo,
				JobID:          jobID,
				QJobID:         1,
				Args:           jobArgs,
				ErrorCount:     riverJob.Attempt,
			})

			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			}

			if tt.expLogMsg != "" {
				assert.Regexp(t, regexp.MustCompile(tt.expLogMsg), hook.LastEntry().Message)
			}

			// Check if job should be acknowledged
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
