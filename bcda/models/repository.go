package models

import (
	"context"
	"time"

	"github.com/pborman/uuid"
)

// Repository contains all of the CRUD methods represented in the models package from the storage layer
type Repository interface {
	cclfFileRepository
	cclfBeneficiaryRepository
	suppressionRepository
	jobRepository
}

type cclfFileRepository interface {
	// GetLatest returns the latest CCLF File (most recent timestamp) that matches the search criteria.
	// The returned CCLF file will fall between the provided time window.
	// If any of the time values equals time.Time (default value), then the time value IS NOT used in the filtering.
	GetLatestCCLFFile(ctx context.Context, cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time, fileType CCLFFileType) (*CCLFFile, error)
}

// CCLFBeneficiaryRepository contains methods need to interact with CCLF Beneficiary data.
type cclfBeneficiaryRepository interface {
	GetCCLFBeneficiaryMBIs(ctx context.Context, cclfFileID uint) ([]string, error)

	GetCCLFBeneficiaries(ctx context.Context, cclfFileID uint, ignoredMBIs []string) ([]*CCLFBeneficiary, error)
}

type suppressionRepository interface {
	GetSuppressedMBIs(ctx context.Context, lookbackDays int) ([]string, error)
}

type jobRepository interface {
	// CreateJob creates a job and returns the id associated with the updated job
	CreateJob(ctx context.Context, j Job) (jobID uint, err error)

	GetJobs(ctx context.Context, acoID uuid.UUID, statuses ...JobStatus) ([]Job, error)

	UpdateJob(ctx context.Context, j Job) error
}
