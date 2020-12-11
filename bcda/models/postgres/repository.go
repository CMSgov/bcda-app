package postgres

import (
	"errors"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/CMSgov/bcda-app/bcda/models"
	"gorm.io/gorm"
)

// Ensure Repository satisfies the interface
var _ models.Repository = &Repository{}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db}
}

func (r *Repository) GetLatestCCLFFile(cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time, fileType models.CCLFFileType) (*models.CCLFFile, error) {

	const (
		queryNoTime     = "aco_cms_id = ? AND cclf_num = ? AND import_status = ? AND type = ?"
		queryLower      = queryNoTime + " AND timestamp >= ?"
		queryUpper      = queryNoTime + " AND timestamp <= ?"
		queryLowerUpper = queryNoTime + " AND timestamp >= ? AND timestamp <= ?"
	)

	var (
		cclfFile models.CCLFFile
		result   *gorm.DB
	)
	if lowerBound.IsZero() && upperBound.IsZero() {
		result = r.db.Where(queryNoTime,
			cmsID, cclfNum, constants.ImportComplete, fileType)
	} else if !lowerBound.IsZero() && upperBound.IsZero() {
		result = r.db.Where(queryLower,
			cmsID, cclfNum, constants.ImportComplete, fileType,
			lowerBound)
	} else if lowerBound.IsZero() && !upperBound.IsZero() {
		result = r.db.Where(queryUpper,
			cmsID, cclfNum, constants.ImportComplete, fileType,
			upperBound)
	} else {
		result = r.db.Where(queryLowerUpper,
			cmsID, cclfNum, constants.ImportComplete, fileType,
			lowerBound, upperBound)
	}

	result = result.Order("timestamp DESC").First(&cclfFile)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &cclfFile, result.Error
}

func (r *Repository) GetCCLFBeneficiaryMBIs(cclfFileID uint) ([]string, error) {
	var mbis []string

	if err := r.db.Table("cclf_beneficiaries").Where("file_id = ?", cclfFileID).Pluck("mbi", &mbis).Error; err != nil {
		return nil, err
	}

	return mbis, nil
}

func (r *Repository) GetCCLFBeneficiaries(cclfFileID uint, ignoredMBIs []string) ([]*models.CCLFBeneficiary, error) {

	const (
		// this is used to get unique ids for de-duplicating MBIs that are listed multiple times in the CCLF8 file
		idQuery = "SELECT id FROM ( SELECT max(id) as id, mbi FROM cclf_beneficiaries WHERE file_id = ? GROUP BY mbi ) as id"
	)
	var beneficiaries []*models.CCLFBeneficiary

	// NOTE: We changed the query that was being used for "old benes"
	// By querying by IDs, we really should not need to also query by the corresponding MBIs as well
	query := r.db.Where("id in (?)", r.db.Raw(idQuery, cclfFileID))

	if len(ignoredMBIs) != 0 {
		query = query.Not("mbi", ignoredMBIs)
	}

	if err := query.Find(&beneficiaries).Error; err != nil {
		return nil, err
	}

	return beneficiaries, nil
}

func (r *Repository) GetSuppressedMBIs(lookbackDays int) ([]string, error) {
	var suppressedMBIs []string

	// #nosec G202
	if err := r.db.Raw(`SELECT DISTINCT s.mbi
	FROM (
		SELECT mbi, MAX(effective_date) max_date
		FROM suppressions
		WHERE (NOW() - interval '`+strconv.Itoa(lookbackDays)+` days') < effective_date AND effective_date <= NOW()
					AND preference_indicator != ''
		GROUP BY mbi
	) h
	JOIN suppressions s ON s.mbi = h.mbi and s.effective_date = h.max_date
	WHERE preference_indicator = 'N'`).Pluck("mbi", &suppressedMBIs).Error; err != nil {
		return nil, err
	}

	return suppressedMBIs, nil
}
