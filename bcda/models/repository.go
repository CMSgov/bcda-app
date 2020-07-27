package models

import (
	"time"
)

// Repository contains all of the CRUD methods represented in the models package from the storage layer
type Repository interface {
	cclfFileRepository
	cclfBeneficiaryRepository
	suppressionRepository
}

type cclfFileRepository interface {
	// GetLatest returns the latest CCLF File (most recent timestamp)
	// that matches the search criteria.
	// The returned CCLF file will fall between the provided time window.
	// If any of the time values equals time.Time (default value), then the time value IS NOT used in the filtering.
	GetLatestCCLFFile(cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time) (*CCLFFile, error)
}

// CCLFBeneficiaryRepository contains methods need to interact with CCLF Beneficiary data.
type cclfBeneficiaryRepository interface {
	GetCCLFBeneficiaryIds(cclfFileID uint) ([]int64, error)

	GetCCLFBeneficiaryMBIs(cclfFileID uint) ([]string, error)

	GetCCLFBeneficiaries(beneIDs []int64, ignoredMBIs []string) ([]*CCLFBeneficiary, error)
}

type suppressionRepository interface {
	GetSuppressedMBIs(lookbackDays int) ([]string, error)
}
