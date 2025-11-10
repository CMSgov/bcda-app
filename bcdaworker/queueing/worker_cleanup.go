package queueing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	msgr "github.com/CMSgov/bcda-app/bcda/slackmessenger"
	"github.com/CMSgov/bcda-app/bcdaworker/cleanup"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"

	"github.com/slack-go/slack"
)

// TODO: Consider moving functions like cleanupJob and archiveExpiring to receiver methods of CleanupJobWorker
type CleanupJobWorker struct {
	river.WorkerDefaults[worker_types.CleanupJobArgs]
	cleanupJob      func(*sql.DB, time.Time, models.JobStatus, models.JobStatus, ...string) error
	archiveExpiring func(*sql.DB, time.Time) error
	db              *sql.DB
}

func NewCleanupJobWorker(db *sql.DB) *CleanupJobWorker {
	return &CleanupJobWorker{
		cleanupJob:      cleanup.CleanupJob,
		archiveExpiring: cleanup.ArchiveExpiring,
		db:              db,
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
	slackToken := conf.GetEnv("SLACK_TOKEN")

	slackClient := slack.New(slackToken)

	msgr.SendSlackMessage(slackClient, msgr.OperationsChannel, fmt.Sprintf("Started Archive and Clean Job Data for %s environment.", environment), "")

	// Cleanup archived jobs: remove job directory and files from archive and update job status to Expired
	if err := w.cleanupJob(w.db, cutoff, models.JobStatusArchived, models.JobStatusExpired, archiveDir, stagingDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupArchArg)))

		msgr.SendSlackMessage(slackClient, msgr.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", msgr.FailureMsg, environment), msgr.Danger)

		return err
	}

	// Cleanup failed jobs: remove job directory and files from failed jobs and update job status to FailedExpired
	if err := w.cleanupJob(w.db, cutoff, models.JobStatusFailed, models.JobStatusFailedExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupFailedArg)))

		msgr.SendSlackMessage(slackClient, msgr.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", msgr.FailureMsg, environment), msgr.Danger)

		return err
	}

	// Cleanup cancelled jobs: remove job directory and files from cancelled jobs and update job status to CancelledExpired
	if err := w.cleanupJob(w.db, cutoff, models.JobStatusCancelled, models.JobStatusCancelledExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupCancelledArg)))

		msgr.SendSlackMessage(slackClient, msgr.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", msgr.FailureMsg, environment), msgr.Danger)

		return err
	}

	// Archive expiring jobs: update job statuses and move files to an inaccessible location
	if err := w.archiveExpiring(w.db, cutoff); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.ArchiveJobFiles)))

		msgr.SendSlackMessage(slackClient, msgr.AlertsChannel, fmt.Sprintf("%s: Archive and Clean Job in %s env.", msgr.FailureMsg, environment), msgr.Danger)

		return err
	}

	msgr.SendSlackMessage(slackClient, msgr.OperationsChannel, fmt.Sprintf("%s: Archive and Clean Job Data for %s env.", msgr.SuccessMsg, environment), msgr.Good)

	return nil
}
