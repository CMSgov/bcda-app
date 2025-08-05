package queueing

import (
	"context"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	slUtls "github.com/CMSgov/bcda-app/bcda/slack"
	"github.com/CMSgov/bcda-app/bcdaworker/cleanup"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"

	"github.com/slack-go/slack"
)

type CleanupJobWorker struct {
	river.WorkerDefaults[worker_types.CleanupJobArgs]
	cleanupJob      func(time.Time, models.JobStatus, models.JobStatus, ...string) error
	archiveExpiring func(time.Time) error
}

func NewCleanupJobWorker() *CleanupJobWorker {
	return &CleanupJobWorker{
		cleanupJob:      cleanup.CleanupJob,
		archiveExpiring: cleanup.ArchiveExpiring,
	}
}

func (w *CleanupJobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.CleanupJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	_, logger := log.SetCtxLogger(ctx, "transaction_id", uuid.New())

	cutoff := getCutOffTime()
	archiveDir := conf.GetEnv("FHIR_ARCHIVE_DIR")
	stagingDir := conf.GetEnv("FHIR_STAGING_DIR")
	payloadDir := conf.GetEnv("FHIR_PAYLOAD_DIR")
	environment := conf.GetEnv("DEPLOYMENT_TARGET")

	params, err := getAWSParams()
	if err != nil {
		logger.Error("Unable to extract Slack Token from parameter store: %+v", err)
		return err
	}

	slackClient := slack.New(params)

	_, _, err = slackClient.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started Archive and Clean Job Data for %s environment.", environment), false),
	)
	if err != nil {
		logger.Error("Error sending notifier start message: %+v", err)
	}

	// Cleanup archived jobs: remove job directory and files from archive and update job status to Expired
	if err := w.cleanupJob(cutoff, models.JobStatusArchived, models.JobStatusExpired, archiveDir, stagingDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupArchArg)))

		slUtls.SendSlackMessage(slackClient, slUtls.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", slUtls.FailureMsg, environment), false)

		return err
	}

	// Cleanup failed jobs: remove job directory and files from failed jobs and update job status to FailedExpired
	if err := w.cleanupJob(cutoff, models.JobStatusFailed, models.JobStatusFailedExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupFailedArg)))

		slUtls.SendSlackMessage(slackClient, slUtls.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", slUtls.FailureMsg, environment), false)

		return err
	}

	// Cleanup cancelled jobs: remove job directory and files from cancelled jobs and update job status to CancelledExpired
	if err := w.cleanupJob(cutoff, models.JobStatusCancelled, models.JobStatusCancelledExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupCancelledArg)))

		slUtls.SendSlackMessage(slackClient, slUtls.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", slUtls.FailureMsg, environment), false)

		return err
	}

	// Archive expiring jobs: update job statuses and move files to an inaccessible location
	if err := w.archiveExpiring(cutoff); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.ArchiveJobFiles)))

		slUtls.SendSlackMessage(slackClient, slUtls.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", slUtls.FailureMsg, environment), false)

		return err
	}

	slUtls.SendSlackMessage(slackClient, slUtls.OperationsChannel, fmt.Sprintf("%s: Archive and Clean Job Data for %s env.", slUtls.SuccessMsg, environment), true)

	return nil
}
