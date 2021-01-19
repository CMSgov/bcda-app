// package repository contains all of the methods needed to interact with the BCDA data
package repository

import (
	"context"
	"errors"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pborman/uuid"
)

type Repository interface {
	acoRepository
	cclfBeneficiaryRepository
	jobRepository
	jobKeyRepository
}

type acoRepository interface {
	GetACOByUUID(ctx context.Context, uuid uuid.UUID) (*models.ACO, error)
}

type cclfBeneficiaryRepository interface {
	GetCCLFBeneficiaryByID(ctx context.Context, id uint) (*models.CCLFBeneficiary, error)
}
type jobRepository interface {
	GetJobByID(ctx context.Context, jobID uint) (*models.Job, error)

	UpdateJobStatus(ctx context.Context, jobID uint, new models.JobStatus) error

	// UpdateJobStatusCheckStatus updates the particular job indicated by the jobID
	// iff the Job's status field matches current.
	UpdateJobStatusCheckStatus(ctx context.Context, jobID uint, current, new models.JobStatus) error

	IncrementCompletedJobCount(ctx context.Context, jobID uint) error
}

type jobKeyRepository interface {
	CreateJobKey(ctx context.Context, jobKey models.JobKey) error

	GetJobKeyCount(ctx context.Context, jobID uint) (int, error)
}

var (
	ErrJobNotUpdated = errors.New("job was not updated, no match found")
	ErrJobNotFound   = errors.New("no job found for given id")
)
