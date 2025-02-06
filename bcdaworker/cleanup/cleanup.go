package cleanup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
)

type CleanupJobArgs struct {
}

type CleanupJobWorker struct {
	river.WorkerDefaults[CleanupJobArgs]
	cleanupJob      func(time.Time, models.JobStatus, models.JobStatus, ...string) error
	archiveExpiring func(time.Time) error
}

func NewCleanupJobWorker() *CleanupJobWorker {
	return &CleanupJobWorker{
		cleanupJob:      cleanupJob,
		archiveExpiring: archiveExpiring,
	}
}

func (args CleanupJobArgs) Kind() string {
	return "CleanupJob"
}

func (w *CleanupJobWorker) Work(ctx context.Context, rjob *river.Job[CleanupJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	_, logger := log.SetCtxLogger(ctx, "transaction_id", uuid.New())

	cutoff := getCutOffTime()
	archiveDir := conf.GetEnv("FHIR_ARCHIVE_DIR")
	stagingDir := conf.GetEnv("FHIR_STAGING_DIR")
	payloadDir := conf.GetEnv("PAYLOAD_DIR")

	// Cleanup archived jobs: remove job directory and files from archive and update job status to Expired
	if err := w.cleanupJob(cutoff, models.JobStatusArchived, models.JobStatusExpired, archiveDir, stagingDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupArchArg)))
		return err
	}

	// Cleanup failed jobs: remove job directory and files from failed jobs and update job status to FailedExpired
	if err := w.cleanupJob(cutoff, models.JobStatusFailed, models.JobStatusFailedExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupFailedArg)))
		return err
	}

	// Cleanup cancelled jobs: remove job directory and files from cancelled jobs and update job status to CancelledExpired
	if err := w.cleanupJob(cutoff, models.JobStatusCancelled, models.JobStatusCancelledExpired, stagingDir, payloadDir); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.CleanupCancelledArg)))
		return err
	}

	// Archive expiring jobs: update job statuses and move files to an inaccessible location
	if err := w.archiveExpiring(cutoff); err != nil {
		logger.Error(errors.Wrap(err, fmt.Sprintf("failed to process job: %s", constants.ArchiveJobFiles)))
		return err
	}

	return nil
}

func getCutOffTime() time.Time {
	cutoff := time.Now().Add(-time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24)))
	return cutoff
}

func archiveExpiring(maxDate time.Time) error {
	log.API.Info("Archiving expiring job files...")

	db := database.Connection
	r := postgres.NewRepository(db)
	jobs, err := r.GetJobsByUpdateTimeAndStatus(context.Background(),
		time.Time{}, maxDate, models.JobStatusCompleted)
	if err != nil {
		log.API.Error(err)
		return err
	}

	var lastJobError error
	for _, j := range jobs {
		id := j.ID
		jobPayloadDir := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), id)
		_, err = os.Stat(jobPayloadDir)
		jobPayloadDirExist := err == nil
		jobArchiveDir := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_ARCHIVE_DIR"), id)

		if jobPayloadDirExist {
			err = os.Rename(jobPayloadDir, jobArchiveDir)
			if err != nil {
				log.API.Error(err)
				lastJobError = err
				continue
			}
		}

		j.Status = models.JobStatusArchived
		err = r.UpdateJob(context.Background(), *j)
		if err != nil {
			log.API.Error(err)
			lastJobError = err
		}
	}

	return lastJobError
}

func cleanupJob(maxDate time.Time, currentStatus, newStatus models.JobStatus, rootDirsToClean ...string) error {
	db := database.Connection
	r := postgres.NewRepository(db)
	jobs, err := r.GetJobsByUpdateTimeAndStatus(context.Background(),
		time.Time{}, maxDate, currentStatus)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		log.API.Infof("No %s job files to clean", currentStatus)
		return nil
	}

	for _, job := range jobs {
		if err := cleanupJobData(job.ID, rootDirsToClean...); err != nil {
			log.API.Errorf("Unable to cleanup directories %s", err)
			continue
		}

		job.Status = newStatus
		err = r.UpdateJob(context.Background(), *job)
		if err != nil {
			log.API.Errorf("Failed to update job status to %s %s", newStatus, err)
			continue
		}

		log.API.WithFields(logrus.Fields{
			"job_began":     job.CreatedAt,
			"files_removed": time.Now(),
			"job_id":        job.ID,
		}).Infof("Files cleaned from %s and job status set to %s", rootDirsToClean, newStatus)
	}

	return nil
}

func cleanupJobData(jobID uint, rootDirs ...string) error {
	for _, rootDirToClean := range rootDirs {
		dir := filepath.Join(rootDirToClean, strconv.FormatUint(uint64(jobID), 10))
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to remove %s because %s", dir, err)
		}
	}

	return nil
}
