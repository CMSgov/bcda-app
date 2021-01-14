package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"

	"github.com/CMSgov/bcda-app/bcda/models"

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
			repository := postgres.NewRepository(db)

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
			repository := postgres.NewRepository(db)

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
			repository := postgres.NewRepository(db)

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
			repository := postgres.NewRepository(db)

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

// TestDuplicateCCLFFileNames validates behavior of the cclf_files schema.
// Therefore, we need to test against the real postgres instance.
func (s *RepositoryTestSuite) TestDuplicateCCLFFileNames() {
	db := database.GetDbConnection()
	defer db.Close()

	repository := postgres.NewRepository(db)
	tests := []struct {
		name     string
		fileName string
		acoIDs   []string
		errMsg   string
	}{
		{"Different ACO ID", uuid.New(), []string{"ACO1", "ACO2"},
			""},
		{"Duplicate ACO ID", uuid.New(), []string{"ACO3", "ACO3"},
			`pq: duplicate key value violates unique constraint "idx_cclf_files_name_aco_cms_id_key"`},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			var (
				err           error
				expectedCount int
			)
			for _, acoID := range tt.acoIDs {
				cclfFile := models.CCLFFile{
					Name:            tt.fileName,
					ACOCMSID:        acoID,
					Timestamp:       time.Now(),
					PerformanceYear: 20,
				}

				if cclfFile.ID, err = repository.CreateCCLFFile(context.Background(), cclfFile); err != nil {
					continue
				}
				expectedCount++
				assert.True(t, cclfFile.ID > 0, "ID should be set!")
				defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), db, cclfFile.ACOCMSID)
			}

			if tt.errMsg != "" {
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			count := len(postgrestest.GetCCLFFilesByName(s.T(), db, tt.fileName))
			assert.True(t, expectedCount > 0, "Files should've been written")
			assert.Equal(t, expectedCount, count)
		})
	}

}

// TestCMSID verifies that we can store and retrieve the CMS_ID as expected
// i.e. the value is not padded with any extra characters
func (s *RepositoryTestSuite) TestCMSID() {
	db := database.GetDbConnection()
	defer db.Close()
	r := postgres.NewRepository(db)

	cmsID := "V001"
	cclfFile := models.CCLFFile{CCLFNum: 1, Name: "someName", ACOCMSID: cmsID, Timestamp: time.Now(), PerformanceYear: 20}
	aco := models.ACO{UUID: uuid.NewUUID(), CMSID: &cmsID, Name: "someName"}
	var err error

	cclfFile.ID, err = r.CreateCCLFFile(context.Background(), cclfFile)
	assert.NoError(s.T(), err)
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), db, cmsID)

	postgrestest.CreateACO(s.T(), db, aco)
	defer postgrestest.DeleteACO(s.T(), db, aco.UUID)

	actualCMSID := *postgrestest.GetACOByUUID(s.T(), db, aco.UUID).CMSID
	assert.Equal(s.T(), cmsID, actualCMSID)

	actualCMSID = postgrestest.GetCCLFFilesByName(s.T(), db, cclfFile.Name)[0].ACOCMSID
	assert.Equal(s.T(), cmsID, actualCMSID)
}

func (s *RepositoryTestSuite) TestCCLFFileType() {
	db := database.GetDbConnection()
	defer db.Close()
	r := postgres.NewRepository(db)

	cmsID := "T9999"

	noType := models.CCLFFile{
		CCLFNum:         8,
		Name:            uuid.New(),
		ACOCMSID:        cmsID,
		Timestamp:       time.Now(),
		PerformanceYear: 20,
	}
	withType := models.CCLFFile{
		CCLFNum:         8,
		Name:            uuid.New(),
		ACOCMSID:        cmsID,
		Timestamp:       time.Now(),
		PerformanceYear: 20,
		Type:            models.FileTypeRunout,
	}
	var err error

	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), db, cmsID)
	noType.ID, err = r.CreateCCLFFile(context.Background(), noType)
	assert.NoError(s.T(), err)

	withType.ID, err = r.CreateCCLFFile(context.Background(), withType)
	assert.NoError(s.T(), err)

	result := postgrestest.GetCCLFFilesByName(s.T(), db, noType.Name)
	assert.Equal(s.T(), 1, len(result))
	assert.Equal(s.T(), noType.Type, result[0].Type)

	result = postgrestest.GetCCLFFilesByName(s.T(), db, withType.Name)
	assert.Equal(s.T(), 1, len(result))
	assert.Equal(s.T(), withType.Type, result[0].Type)
}

func getCCLFFile(cclfNum int, cmsID, importStatus string, fileType models.CCLFFileType) *models.CCLFFile {
	createTime := time.Now()
	return &models.CCLFFile{
		ID:              uint(rand.Int63()),
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
		ID:           uint(rand.Int63()),
		FileID:       uint(rand.Uint32()),
		MBI:          fmt.Sprintf("MBI%d", rand.Uint32()),
		BlueButtonID: fmt.Sprintf("BlueButton%d", rand.Uint32()),
	}
}
