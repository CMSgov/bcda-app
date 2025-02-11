package cleanup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
)

func ArchiveExpiring(maxDate time.Time) error {
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

func CleanupJob(maxDate time.Time, currentStatus, newStatus models.JobStatus, rootDirsToClean ...string) error {
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
