package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/CMSgov/bcda-app/bcda/models"
	"gorm.io/gorm"
)

const (
	sqlFlavor = sqlbuilder.PostgreSQL
)

// Ensure Repository satisfies the interface
var _ models.Repository = &Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *gorm.DB) *Repository {
	// FIXME: Take *sql.DB instead
	db1, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	return &Repository{db1}
}

func (r *Repository) GetLatestCCLFFile(cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time, fileType models.CCLFFileType) (*models.CCLFFile, error) {

	const (
		queryNoTime     = "aco_cms_id = ? AND cclf_num = ? AND import_status = ? AND type = ?"
		queryLower      = queryNoTime + " AND timestamp >= ?"
		queryUpper      = queryNoTime + " AND timestamp <= ?"
		queryLowerUpper = queryNoTime + " AND timestamp >= ? AND timestamp <= ?"
	)

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "name", "timestamp", "performance_year")
	sb.From("cclf_files")
	sb.Where(
		sb.Equal("aco_cms_id", cmsID),
		sb.Equal("cclf_num", cclfNum),
		sb.Equal("import_status", importStatus),
		sb.Equal("type", fileType),
	)

	cclfFile := models.CCLFFile{
		ACOCMSID:     cmsID,
		CCLFNum:      cclfNum,
		ImportStatus: importStatus,
		Type:         fileType,
	}

	if !lowerBound.IsZero() && upperBound.IsZero() {
		sb.Where(sb.GreaterEqualThan("timestamp", lowerBound))
	} else if lowerBound.IsZero() && !upperBound.IsZero() {
		sb.Where(sb.LessEqualThan("timestamp", upperBound))
	} else if !lowerBound.IsZero() && !upperBound.IsZero() {
		sb.Where(
			sb.GreaterEqualThan("timestamp", lowerBound),
			sb.LessEqualThan("timestamp", upperBound),
		)
	}
	sb.OrderBy("timestamp").Desc().Limit(1)

	query, args := sb.Build()
	row := r.db.QueryRow(query, args...)
	if err := row.Scan(&cclfFile.ID, &cclfFile.Name, &cclfFile.Timestamp, &cclfFile.PerformanceYear); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &cclfFile, nil
}

func (r *Repository) GetCCLFBeneficiaryMBIs(cclfFileID uint) ([]string, error) {
	var mbis []string

	sb := sqlFlavor.NewSelectBuilder().Select("mbi").From("cclf_beneficiaries")
	sb.Where(sb.Equal("file_id", cclfFileID))

	query, args := sb.Build()
	rows, err := r.db.Query(query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mbi string
		if err = rows.Scan(&mbi); err != nil {
			return nil, err
		}
		mbis = append(mbis, mbi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return mbis, nil
}

func (r *Repository) GetCCLFBeneficiaries(cclfFileID uint, ignoredMBIs []string) ([]*models.CCLFBeneficiary, error) {
	var beneficiaries []*models.CCLFBeneficiary

	// Subquery to deal with duplicate MBIs found within a single CCLF file.
	// NOTE: We no longer have duplicate MBIs after this PR: https://github.com/CMSgov/bcda-app/pull/583
	// We have to remove duplicates on older files, but once that's done, we can remove the subquery
	// and query for the benes by file_id directly.
	subSB := sqlFlavor.NewSelectBuilder()
	subSB.Select("MAX(id)").From("cclf_beneficiaries").Where(
		subSB.Equal("file_id", cclfFileID),
	).GroupBy("mbi")

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "file_id", "mbi", "blue_button_id")
	sb.From("cclf_beneficiaries").Where(sb.In("id", subSB))

	if len(ignoredMBIs) != 0 {
		ignored := make([]interface{}, len(ignoredMBIs))
		for i, v := range ignoredMBIs {
			ignored[i] = v
		}
		sb.Where(sb.NotIn("mbi", ignored...))
	}

	query, args := sb.Build()
	rows, err := r.db.Query(query, args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bene models.CCLFBeneficiary
		if err := rows.Scan(&bene.ID, &bene.FileID, &bene.MBI, &bene.BlueButtonID); err != nil {
			return nil, err
		}
		beneficiaries = append(beneficiaries, &bene)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return beneficiaries, nil
}

func (r *Repository) GetSuppressedMBIs(lookbackDays int) ([]string, error) {
	var suppressedMBIs []string

	subSB := sqlFlavor.NewSelectBuilder()
	subSB.Select("mbi", "MAX(effective_date) as max_date").From("suppressions")
	subSB.Where(
		subSB.Between("effective_date", sqlbuilder.Raw(fmt.Sprintf("NOW() - interval '%d days'", lookbackDays)), sqlbuilder.Raw("NOW()")),
		subSB.NotEqual("preference_indicator", ""),
	).GroupBy("mbi")

	sb := sqlFlavor.NewSelectBuilder().Distinct().Select("s.mbi")
	sb.From(sb.BuilderAs(subSB, "h")).Join("suppressions s", "s.mbi = h.mbi", "s.effective_date = h.max_date")
	sb.Where(sb.Equal("preference_indicator", "N"))

	query, args := sb.Build()
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mbi string
		if err = rows.Scan(&mbi); err != nil {
			return nil, err
		}
		suppressedMBIs = append(suppressedMBIs, mbi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return suppressedMBIs, nil
}
