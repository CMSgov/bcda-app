package queueing

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/log"
	"github.com/bgentry/que-go"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProcessJobFailedValidation(t *testing.T) {
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
			var err error
			worker := &worker.MockWorker{}
			defer worker.AssertExpectations(t)

			repo := repository.NewMockRepository(t)
			queue := &queue{worker: worker, repository: repo, log: logger}

			id, err := safecast.ToUint(1)
			if err != nil {
				t.Fatal(err)
			}
			job := models.Job{ID: id}

			jobid, e := safecast.ToInt(1)
			if e != nil {
				t.Fatal(e)
			}
			jobArgs := models.JobEnqueueArgs{ID: jobid, ACOID: uuid.New()}

			queJob := que.Job{ID: 1}
			queJob.Args, err = json.Marshal(jobArgs)
			assert.NoError(t, err)

			// Set the error count to max to ensure that we've exceeded the retries
			if tt.name == "NoParentJobRetriesExceeded" {
				queJob.ErrorCount = testUtils.CryptoRandInt31()
			}

			worker.On("ValidateJob", testUtils.CtxMatcher, int64(1), jobArgs).Return(nil, tt.validateErr)

			if tt.name == "QueJobAlreadyProcessed" {
				job.Status = models.JobStatusCompleted
				repo.On("GetJobByID", testUtils.CtxMatcher, job.ID).Return(&job, nil)
			}

			err = queue.processJob(&queJob)
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			}

			if tt.expLogMsg != "" {
				assert.Regexp(t, regexp.MustCompile(tt.expLogMsg), hook.LastEntry().Message)
			}
		})
	}

}

func TestCheckIfCancelled(t *testing.T) {
	q := MasterQueue{
		queue: &queue{
			repository: nil,
		},
		StagingDir: "",
		PayloadDir: "",
		MaxRetry:   0,
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
	jobs := models.JobEnqueueArgs{}

	jobID, err := safecast.ToInt64(jobs.ID)
	assert.NoError(t, err)

	// In produation we wait 15 second intervals, for test we do 1
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
