package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/CMSgov/bcda-app/bcda/models"
	"gorm.io/gorm"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
	suite.Suite
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

func (r *RepositoryTestSuite) TestGetLatestCCLFFile() {
	cmsID := "cmsID"
	cclfNum := int(8)
	importStatus := constants.ImportComplete

	tests := []struct {
		name          string
		lowerBound    time.Time
		upperBound    time.Time
		fileType      models.CCLFFileType
		expQueryRegex string
		result        *models.CCLFFile
	}{
		{
			"NoTime",
			time.Time{},
			time.Time{},
			models.FileTypeDefault,
			`SELECT id, name, timestamp, performance_year FROM cclf_files WHERE aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND type = $4 ORDER BY timestamp DESC LIMIT 1`,
			getCCLFFile(cclfNum, cmsID, importStatus, models.FileTypeDefault),
		},
		{
			"Runout",
			time.Time{},
			time.Time{},
			models.FileTypeRunout,
			`SELECT id, name, timestamp, performance_year FROM cclf_files WHERE aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND type = $4 ORDER BY timestamp DESC LIMIT 1`,
			getCCLFFile(cclfNum, cmsID, importStatus, models.FileTypeRunout),
		},
		{
			"LowerBoundTime",
			time.Now(),
			time.Time{},
			models.FileTypeDefault,
			`SELECT id, name, timestamp, performance_year FROM cclf_files WHERE aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND type = $4 AND timestamp >= $5 ORDER BY timestamp DESC LIMIT 1`,
			getCCLFFile(cclfNum, cmsID, importStatus, models.FileTypeDefault),
		},
		{
			"UpperBoundTime",
			time.Time{},
			time.Now(),
			models.FileTypeDefault,
			`SELECT id, name, timestamp, performance_year FROM cclf_files WHERE aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND type = $4 AND timestamp <= $5 ORDER BY timestamp DESC LIMIT 1`,
			getCCLFFile(cclfNum, cmsID, importStatus, models.FileTypeDefault),
		},
		{
			"LowerAndUpperBoundTime",
			time.Now(),
			time.Now(),
			models.FileTypeDefault,
			`SELECT id, name, timestamp, performance_year FROM cclf_files WHERE aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND type = $4 AND timestamp >= $5 AND timestamp <= $6 ORDER BY timestamp DESC LIMIT 1`,
			getCCLFFile(cclfNum, cmsID, importStatus, models.FileTypeDefault),
		},
		{
			"NoResult",
			time.Time{},
			time.Time{},
			models.FileTypeDefault,
			`SELECT id, name, timestamp, performance_year FROM cclf_files WHERE aco_cms_id = $1 AND cclf_num = $2 AND import_status = $3 AND type = $4 ORDER BY timestamp DESC LIMIT 1`,
			nil,
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {

			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			}()
			repository := NewRepository(db)

			args := []driver.Value{cmsID, cclfNum, importStatus, tt.fileType}
			if !tt.lowerBound.IsZero() {
				args = append(args, tt.lowerBound)
			}
			if !tt.upperBound.IsZero() {
				args = append(args, tt.upperBound)
			}

			query := mock.ExpectQuery(fmt.Sprintf("^%s$", regexp.QuoteMeta(tt.expQueryRegex))).
				WithArgs(args...)
			if tt.result == nil {
				query.WillReturnError(sql.ErrNoRows)
			} else {
				query.WillReturnRows(sqlmock.
					NewRows([]string{"id", "name", "timestamp", "performance_year"}).
					AddRow(tt.result.ID, tt.result.Name, tt.result.Timestamp, tt.result.PerformanceYear))
			}
			cclfFile, err := repository.GetLatestCCLFFile(context.Background(), cmsID, cclfNum, importStatus, tt.lowerBound, tt.upperBound,
				tt.fileType)
			assert.NoError(t, err)

			if tt.result == nil {
				assert.Nil(t, cclfFile)
			} else {
				assert.Equal(t, tt.result, cclfFile)
			}
		})
	}
}

func (r *RepositoryTestSuite) TestGetCCLFBeneficiaryMBIs() {
	tests := []struct {
		name          string
		expQueryRegex string
		errToReturn   error
	}{
		{
			"HappyPath",
			`SELECT mbi FROM cclf_beneficiaries WHERE file_id = $1`,
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT mbi FROM cclf_beneficiaries WHERE file_id = $1`,
			fmt.Errorf("Some SQL error"),
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
			mbis := []string{"0", "1", "2"}
			cclfFileID := uint(rand.Int63())

			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			}()
			repository := NewRepository(db)

			query := mock.ExpectQuery(fmt.Sprintf("^%s$", regexp.QuoteMeta(tt.expQueryRegex))).
				WithArgs(cclfFileID)
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"mbi"})
				for _, mbi := range mbis {
					rows.AddRow(mbi)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetCCLFBeneficiaryMBIs(context.Background(), cclfFileID)
			if tt.errToReturn == nil {
				assert.NoError(t, err)
				assert.Equal(t, mbis, result)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func (r *RepositoryTestSuite) TestGetCCLFBeneficiaries() {
	tests := []struct {
		name            string
		expQueryRegex   string
		ignoredMBIs     []string
		expectedResults []*models.CCLFBeneficiary
		errToReturn     error
	}{
		{
			"NoIgnoreMBIs",
			`SELECT id, file_id, mbi, blue_button_id FROM cclf_beneficiaries WHERE id IN (SELECT MAX(id) FROM cclf_beneficiaries WHERE file_id = $1 GROUP BY mbi)`,
			nil,
			[]*models.CCLFBeneficiary{
				getCCLFBeneficiary(),
				getCCLFBeneficiary(),
				getCCLFBeneficiary(),
				getCCLFBeneficiary(),
			},
			nil,
		},
		{
			"IgnoredMBIs",
			`SELECT id, file_id, mbi, blue_button_id FROM cclf_beneficiaries WHERE id IN (SELECT MAX(id) FROM cclf_beneficiaries WHERE file_id = $1 GROUP BY mbi) AND mbi NOT IN ($2, $3)`,
			[]string{"123", "456"},
			[]*models.CCLFBeneficiary{
				getCCLFBeneficiary(),
			},
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT id, file_id, mbi, blue_button_id FROM cclf_beneficiaries WHERE id IN (SELECT MAX(id) FROM cclf_beneficiaries WHERE file_id = $1 GROUP BY mbi)`,
			nil,
			nil,
			fmt.Errorf("Some SQL error"),
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
			cclfFileID := uint(rand.Int63())

			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			}()
			repository := NewRepository(db)

			var query *sqlmock.ExpectedQuery
			if tt.ignoredMBIs == nil {
				query = mock.ExpectQuery(fmt.Sprintf("^%s$", regexp.QuoteMeta(tt.expQueryRegex))).
					WithArgs(cclfFileID)
			} else {
				args := []driver.Value{cclfFileID}
				for _, ignoredMBI := range tt.ignoredMBIs {
					args = append(args, ignoredMBI)
				}
				query = mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex)).
					WithArgs(args...)
			}
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"id", "file_id", "mbi", "blue_button_id"})
				for _, bene := range tt.expectedResults {
					rows.AddRow(bene.ID, bene.FileID, bene.MBI, bene.BlueButtonID)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetCCLFBeneficiaries(context.Background(), cclfFileID, tt.ignoredMBIs)
			if tt.errToReturn == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResults, result)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func (r *RepositoryTestSuite) TestGetSuppressedMBIs() {
	lookbackDays := 10
	tests := []struct {
		name          string
		expQueryRegex string
		errToReturn   error
	}{
		{
			"HappyPath",
			`SELECT DISTINCT s.mbi FROM (SELECT mbi, MAX(effective_date) as max_date FROM suppressions WHERE effective_date BETWEEN NOW() - interval '10 days' AND NOW()  AND preference_indicator <> $1 GROUP BY mbi) AS h JOIN suppressions s ON s.mbi = h.mbi AND s.effective_date = h.max_date WHERE preference_indicator = $2`,
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT DISTINCT s.mbi FROM (SELECT mbi, MAX(effective_date) as max_date FROM suppressions WHERE effective_date BETWEEN NOW() - interval '10 days' AND NOW()  AND preference_indicator <> $1 GROUP BY mbi) AS h JOIN suppressions s ON s.mbi = h.mbi AND s.effective_date = h.max_date WHERE preference_indicator = $2`,
			fmt.Errorf("Some SQL error"),
		},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
			suppressedMBIs := []string{"0", "1", "2"}
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			}()
			repository := NewRepository(db)

			// No arguments because the lookback days is embedded in the query
			query := mock.ExpectQuery(regexp.QuoteMeta(tt.expQueryRegex)).WithArgs("", "N")
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"mbi"})
				for _, mbi := range suppressedMBIs {
					rows.AddRow(mbi)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetSuppressedMBIs(context.Background(), lookbackDays)
			if tt.errToReturn == nil {
				assert.NoError(t, err)
				assert.Equal(t, suppressedMBIs, result)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func getCCLFFile(cclfNum int, cmsID, importStatus string, fileType models.CCLFFileType) *models.CCLFFile {
	createTime := time.Now()
	return &models.CCLFFile{
		Model: gorm.Model{
			ID: uint(rand.Int63()),
		},
		CCLFNum:         cclfNum,
		Name:            fmt.Sprintf("CCLFFile%d", rand.Uint64()),
		ACOCMSID:        cmsID,
		Timestamp:       createTime,
		PerformanceYear: 2020,
		ImportStatus:    importStatus,
		Type:            fileType,
	}
}

func getCCLFBeneficiary() *models.CCLFBeneficiary {
	return &models.CCLFBeneficiary{
		Model: gorm.Model{
			ID: uint(rand.Int63()),
		},
		FileID:       uint(rand.Uint32()),
		MBI:          fmt.Sprintf("MBI%d", rand.Uint32()),
		BlueButtonID: fmt.Sprintf("BlueButton%d", rand.Uint32()),
	}
}
