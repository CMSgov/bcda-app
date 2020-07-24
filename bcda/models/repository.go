package models

import (
	"time"
)

// CCLFFileRepository contains methods need to interact with CCLF files
type CCLFFileRepository interface {
	// GetLatest returns the latest CCLF File (most recent timestamp)
	// that matches the search criteria.
	// The returned CCLF file will fall between the provided time window.
	// If any of the time values equals time.Time (default value), then the time value IS NOT used in the filtering.
	GetLatestCCLFFile(cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time) (*CCLFFile, error)
}

// CCLFBeneficiaryRepository contains methods need to interact with CCLF Beneficiary data.
type CCLFBeneficiaryRepository interface {
	GetCCLFBeneficiaryIds(cclfFileID string) ([]int64, error)

	GetCCLFBeneficiaryMBIs(cclfFileID string) ([]string, error)

	GetCCLFBeneficiaries(cclfFileID string, ignoredMBIs []string, ids []int64) ([]*CCLFBeneficiary, error)
}

type SuppressionRepository interface {
	GetSuppressedMBIs(lookbackDays int) ([]string, error)
}
