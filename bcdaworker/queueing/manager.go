package queueing

import (
	"context"
	"database/sql"
	goerrors "errors"
	"fmt"
	"time"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/ccoveille/go-safecast"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
)

// manager is designed to be the shared space for common logic and structs between
// the specific queue library implementations

// queue is responsible for setting the required components necessary for queue clients
// to pick up and handle work
type queue struct {
	worker worker.Worker

	// Resources associated with River library
	client *river.Client[pgxv5.Tx]

	ctx        context.Context
	repository repository.Repository
}

type ValidateJobConfig struct {
	WorkerInstance worker.Worker
	Logger         logrus.FieldLogger
	Repository     repository.Repository
	JobID          int64
	QJobID         int64
	Args           worker_types.JobEnqueueArgs
	ErrorCount     int
}

// This is a weird function as it seems mostly unnecessary.  Could all of this logic just live in worker.ValidateJob?
// On top of that we are checking for each type of error return and saying that some are allowed and should
// acknowledge the job as successful but without doing anything (effectively just removes it from the queue).
// The third return bool param (on true) allows us to succeed (acknowledge) a job.
func validateJob(ctx context.Context, cfg ValidateJobConfig) (*models.Job, error, bool) {
	exportJob, err := cfg.WorkerInstance.ValidateJob(ctx, cfg.QJobID, cfg.Args)

	if goerrors.Is(err, worker.ErrParentJobCancelled) {
		// ACK the job because we do not need to work on queue jobs associated with a cancelled parent job
		cfg.Logger.Warnf("QJob %d associated with a cancelled parent Job %d. Removing job from queue.", cfg.Args.ID, cfg.JobID)
		return nil, nil, true
	} else if goerrors.Is(err, worker.ErrParentJobFailed) {
		// ACK the job because we do not need to work on queue jobs associated with a failed parent job
		cfg.Logger.Warnf("QJob %d associated with a failed parent Job %d. Removing job from queue.", cfg.Args.ID, cfg.JobID)
		return nil, nil, true
	} else if goerrors.Is(err, worker.ErrNoBasePathSet) {
		// Data is corrupted, we cannot work on this job.
		cfg.Logger.Warnf("QJob %d does not contain valid base path. Removing job from queue.", cfg.JobID)
		return nil, nil, true
	} else if goerrors.Is(err, worker.ErrParentJobNotFound) {
		// Based on the current backoff delay (j.ErrorCount^4 + 3 seconds), this should've given
		// us plenty of headroom to ensure that the parent job will never be found.
		maxNotFoundRetries, err := safecast.ToInt(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))
		if err != nil {
			cfg.Logger.Errorf("Failed to convert BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES to int32. Defaulting to 3. Error: %s", err)
			return nil, err, true
		}

		if cfg.ErrorCount >= maxNotFoundRetries {
			cfg.Logger.Errorf("No job found for Job: %d acoID: %s. Retries exhausted. Removing job from queue.", cfg.JobID, cfg.Args.ACOID)
			return nil, nil, true
		}

		cfg.Logger.Warnf("No job found for Job: %d acoID: %s. Will retry.", cfg.JobID, cfg.Args.ACOID)

		return nil, errors.Wrap(repository.ErrJobNotFound, "could not retrieve job from database"), false
	} else if goerrors.Is(err, worker.ErrQueJobProcessed) {
		cfg.Logger.Warnf("QJob %d already processed for parent Job: %d. Checking completion status and removing job from queue.", cfg.Args.ID, cfg.JobID)

		u, err := safecast.ToUint(cfg.JobID)
		if err != nil {
			cfg.Logger.Errorf("Failed to convert Job ID to uint. Error: %s", err)
			return nil, err, true
		}

		_, err = worker.CheckJobCompleteAndCleanup(ctx, cfg.Repository, u)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Error checking job completion & cleanup for Job %d", cfg.JobID)), false
		}

		return nil, nil, true
	} else if err != nil {
		err := errors.Wrap(err, "Failed to validate job")
		cfg.Logger.Error(err)
		return nil, err, false
	}

	return exportJob, err, false
}

func checkIfCancelled(
	ctx context.Context,
	r repository.Repository,
	cancel context.CancelFunc,
	jobID int64,
	wait uint8,
) {
	newID, err := safecast.ToUint(jobID)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-time.After(time.Duration(wait) * time.Second):
			jobStatus, err := r.GetJobByID(ctx, newID)

			if err != nil {
				log.Worker.Warnf("Could not find job %d status: %s", newID, err)
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

// Update the AWS Cloudwatch Metric for job queue count
func updateJobQueueCountCloudwatchMetric(ctx context.Context, db *sql.DB, log logrus.FieldLogger) {
	cloudWatchEnv := conf.GetEnv("DEPLOYMENT_TARGET")
	if cloudWatchEnv != "" {
		err := bcdaaws.PutMetricSample(
			ctx,
			"JobQueueCount",
			"BCDA",
			"Count",
			getQueueJobCount(db, log),
			[]types.Dimension{{Name: aws.String("Environment"), Value: aws.String(cloudWatchEnv)}},
		)
		if err != nil {
			log.Error(err)
		}
	}
}

func getQueueJobCount(db *sql.DB, log logrus.FieldLogger) float64 {
	row := db.QueryRow(`SELECT COUNT(*) FROM river_job WHERE state NOT IN ('completed', 'cancelled', 'discarded');`)

	var count int
	if err := row.Scan(&count); err != nil {
		log.Error(err)
	}

	return float64(count)
}
