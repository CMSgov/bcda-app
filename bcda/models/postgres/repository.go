package postgres

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jinzhu/gorm"
)

type Repository struct {
	db *gorm.DB
}

func (r *Repository) GetLatestCCLFFile(cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time) (*models.CCLFFile, error) {

	const (
		queryNoTime     = "aco_cms_id = ? AND cclf_num = ? AND import_status = ?"
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
			cmsID, cclfNum, constants.ImportComplete).First(&cclfFile)
	} else if !lowerBound.IsZero() && upperBound.IsZero() {
		result = r.db.Where(queryLower,
			cmsID, cclfNum, constants.ImportComplete,
			lowerBound).First(&cclfFile).First(&cclfFile)
	} else if lowerBound.IsZero() && !upperBound.IsZero() {
		result = r.db.Where(queryUpper,
			cmsID, cclfNum, constants.ImportComplete,
			upperBound).First(&cclfFile)
	} else {
		result = r.db.Where(queryLowerUpper,
			cmsID, cclfNum, constants.ImportComplete,
			lowerBound, upperBound).First(&cclfFile)
	}

	if result.RecordNotFound() {
		return nil, nil
	}

	return &cclfFile, result.Error
}

func (r *Repository) GetCCLFBeneficiaryIds(cclfFileID uint) ([]int64, error) {
	var uniqueIds []int64
	if err := r.db.Raw("SELECT id FROM ( SELECT max(id) as id, mbi FROM cclf_beneficiaries WHERE file_id = ? GROUP BY mbi ) as id", cclfFileID).Pluck("id", &uniqueIds).Error; err != nil {
		return nil, err
	}

	return uniqueIds, nil
}

func (r *Repository) GetCCLFBeneficiaryMBIs(cclfFileID uint) ([]string, error) {
	var mbis []string

	if err := r.db.Table("cclf_beneficiaries").Where("file_id = ?", cclfFileID).Pluck("mbi", &mbis).Error; err != nil {
		return nil, err
	}

	return mbis, nil
}

func (r *Repository) GetCCLFBeneficiaries(beneIDs []int64, ignoredMBIs []string) ([]*models.CCLFBeneficiary, error) {

	var beneficiaries []*models.CCLFBeneficiary

	// NOTE: We changed the query that was being used for "old benes"
	// By querying by IDs, we really should not need to also query by the corresponding MBIs as well
	query := r.db.Where("id in (?)", beneIDs)

	if len(ignoredMBIs) != 0 {
		query.Not("mbi", ignoredMBIs)
	}

	if err := r.db.Find(&beneficiaries).Error; err != nil {
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
		WHERE (NOW() - interval '? days') < effective_date AND effective_date <= NOW()
					AND preference_indicator != ''
		GROUP BY mbi
	) h
	JOIN suppressions s ON s.mbi = h.mbi and s.effective_date = h.max_date
	WHERE preference_indicator = 'N'`, lookbackDays).Pluck("mbi", &suppressedMBIs).Error; err != nil {
		return nil, err
	}

	return suppressedMBIs, nil
}
