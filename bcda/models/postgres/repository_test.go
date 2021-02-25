package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"

	"github.com/CMSgov/bcda-app/bcda/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
	suite.Suite

	db         *sql.DB
	repository *postgres.Repository
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

func (r *RepositoryTestSuite) SetupSuite() {
	r.db = database.GetDbConnection()
	r.repository = postgres.NewRepository(r.db)
}

func (r *RepositoryTestSuite) TearDownSuite() {
	r.db.Close()
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
		upperBound    time.Time
		errToReturn   error
	}{
		{
			"HappyPath",
			`SELECT DISTINCT s.mbi FROM (SELECT mbi, MAX(effective_date) as max_date FROM suppressions WHERE effective_date BETWEEN NOW() - interval '10 days' AND NOW() AND preference_indicator <> $1 GROUP BY mbi) AS h JOIN suppressions s ON s.mbi = h.mbi AND s.effective_date = h.max_date WHERE preference_indicator = $2`,
			time.Time{},
			nil,
		},
		{
			"WithUpperBound",
			`SELECT DISTINCT s.mbi FROM (SELECT mbi, MAX(effective_date) as max_date FROM suppressions WHERE effective_date BETWEEN NOW() - interval '10 days' AND NOW() AND preference_indicator <> $1 GROUP BY mbi) AS h JOIN suppressions s ON s.mbi = h.mbi AND s.effective_date = h.max_date WHERE preference_indicator = $2 AND effective_date <= $3`,
			time.Now().Add(-1 * 24 * time.Hour),
			nil,
		},
		{
			"ErrorOnQuery",
			`SELECT DISTINCT s.mbi FROM (SELECT mbi, MAX(effective_date) as max_date FROM suppressions WHERE effective_date BETWEEN NOW() - interval '10 days' AND NOW()  AND preference_indicator <> $1 GROUP BY mbi) AS h JOIN suppressions s ON s.mbi = h.mbi AND s.effective_date = h.max_date WHERE preference_indicator = $2`,
			time.Time{},
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

			args := []driver.Value{"", "N"}

			if !tt.upperBound.IsZero() {
				args = append(args, tt.upperBound)
			}

			query := mock.ExpectQuery(fmt.Sprintf("^%s$", regexp.QuoteMeta(tt.expQueryRegex))).
				WithArgs(args...)
			if tt.errToReturn == nil {
				rows := sqlmock.NewRows([]string{"mbi"})
				for _, mbi := range suppressedMBIs {
					rows.AddRow(mbi)
				}
				query.WillReturnRows(rows)
			} else {
				query.WillReturnError(tt.errToReturn)
			}

			result, err := repository.GetSuppressedMBIs(context.Background(), lookbackDays, tt.upperBound)
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
func (r *RepositoryTestSuite) TestDuplicateCCLFFileNames() {
	tests := []struct {
		name     string
		fileName string
		acoIDs   []string
		errMsg   string
	}{
		{"Different ACO ID", uuid.New(), []string{"ACO1", "ACO2"},
			""},
		{"Duplicate ACO ID", uuid.New(), []string{"ACO3", "ACO3"},
			`duplicate key value violates unique constraint "idx_cclf_files_name_aco_cms_id_key"`},
	}

	for _, tt := range tests {
		r.T().Run(tt.name, func(t *testing.T) {
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

				if cclfFile.ID, err = r.repository.CreateCCLFFile(context.Background(), cclfFile); err != nil {
					continue
				}
				expectedCount++
				assert.True(t, cclfFile.ID > 0, "ID should be set!")
				defer postgrestest.DeleteCCLFFilesByCMSID(r.T(), r.db, cclfFile.ACOCMSID)
			}

			if tt.errMsg != "" {
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			count := len(postgrestest.GetCCLFFilesByName(r.T(), r.db, tt.fileName))
			assert.True(t, expectedCount > 0, "Files should've been written")
			assert.Equal(t, expectedCount, count)
		})
	}
}

// TestACOMethods validates the CRUD operations associated with the acos table
func (r *RepositoryTestSuite) TestACOMethods() {
	assert := r.Assert()

	now := time.Now().UTC().Round(time.Millisecond)
	termination := &models.Termination{
		TerminationDate:     now,
		CutoffDate:          now.Add(time.Hour).Round(time.Millisecond),
		BlacklistType:       models.Voluntary,
		AttributionStrategy: models.AttributionHistorical,
		OptOutStrategy:      models.OptOutLatest,
		ClaimsStrategy:      models.ClaimsHistorical,
	}

	ctx := context.Background()
	cmsID := testUtils.RandomHexID()[0:4]
	terminatedCMSID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{UUID: uuid.NewRandom(), Name: uuid.New(), ClientID: uuid.New(), CMSID: &cmsID}
	terminatedACO := models.ACO{UUID: uuid.NewRandom(), Name: uuid.New(), ClientID: uuid.New(), CMSID: &terminatedCMSID,
		TerminationDetails: termination}

	assert.NoError(r.repository.CreateACO(ctx, aco))
	assert.NoError(r.repository.CreateACO(ctx, terminatedACO))

	defer func() {
		postgrestest.DeleteACO(r.T(), r.db, aco.UUID)
		postgrestest.DeleteACO(r.T(), r.db, terminatedACO.UUID)
	}()

	// Load the ID to allow us to compare entire ACO entities
	aco.ID = postgrestest.GetACOByCMSID(r.T(), r.db, cmsID).ID
	terminatedACO.ID = postgrestest.GetACOByCMSID(r.T(), r.db, terminatedCMSID).ID

	aco.Blacklisted = true
	assert.NoError(r.repository.UpdateACO(ctx, aco.UUID,
		map[string]interface{}{"blacklisted": aco.Blacklisted}))

	res, err := r.repository.GetACOByCMSID(ctx, cmsID)
	assert.NoError(err)
	assert.Equal(aco, *res)

	res, err = r.repository.GetACOByClientID(ctx, aco.ClientID)
	assert.NoError(err)
	assert.Equal(aco, *res)

	res, err = r.repository.GetACOByUUID(ctx, aco.UUID)
	assert.NoError(err)
	assert.Equal(aco, *res)

	res, err = r.repository.GetACOByCMSID(ctx, terminatedCMSID)
	assert.NoError(err)
	assert.Equal(terminatedACO, *res)

	res, err = r.repository.GetACOByClientID(ctx, terminatedACO.ClientID)
	assert.NoError(err)
	assert.Equal(terminatedACO, *res)

	res, err = r.repository.GetACOByUUID(ctx, terminatedACO.UUID)
	assert.NoError(err)
	assert.Equal(terminatedACO, *res)

	// Negative cases
	res, err = r.repository.GetACOByCMSID(ctx, aco.UUID.String())
	assert.EqualError(err, "no ACO record found for "+aco.UUID.String())
	assert.Nil(res)

	res, err = r.repository.GetACOByClientID(ctx, aco.UUID.String())
	assert.EqualError(err, "no ACO record found for "+aco.UUID.String())
	assert.Nil(res)

	res, err = r.repository.GetACOByUUID(ctx, uuid.Parse(aco.ClientID))
	assert.EqualError(err, "no ACO record found for "+aco.ClientID)
	assert.Nil(res)

	assert.Contains(
		r.repository.UpdateACO(ctx, aco.UUID,
			map[string]interface{}{"some_unknown_column": uuid.New()}).Error(),
		"column \"some_unknown_column\" of relation \"acos\" does not exist")
	assert.EqualError(
		r.repository.UpdateACO(ctx, uuid.Parse(aco.ClientID),
			map[string]interface{}{"blacklisted": true}),
		fmt.Sprintf("ACO %s not updated, no row found", aco.ClientID))
	assert.Contains(r.repository.CreateACO(ctx, aco).Error(), "duplicate key value violates unique constraint \"acos_cms_id_key\"")
}

// TestCCLFFilesMethods validates the CRUD operations associated with the cclf_files table
func (r *RepositoryTestSuite) TestCCLFFilesMethods() {
	var err error
	cmsID := testUtils.RandomHexID()[0:4]
	ctx := context.Background()
	assert := r.Assert()

	// Since we have a foreign key tie, we need the cclf file to exist before creating associated benes
	cclfFileFailed := *getCCLFFile(8, cmsID, "Failed", models.FileTypeDefault)
	cclfFileSuccess := *getCCLFFile(8, cmsID, "Success", models.FileTypeDefault)
	cclfFileSuccessOld := *getCCLFFile(8, cmsID, "Success", models.FileTypeDefault)
	cclfFileSuccessOld.Timestamp = cclfFileFailed.Timestamp.Add(-24 * time.Hour)
	cclfFileOther := *getCCLFFile(6, cmsID, "Other", models.FileTypeRunout)

	defer postgrestest.DeleteCCLFFilesByCMSID(r.T(), r.db, cmsID)

	cclfFileFailed.ID, err = r.repository.CreateCCLFFile(ctx, cclfFileFailed)
	assert.NoError(err)
	cclfFileSuccess.ID, err = r.repository.CreateCCLFFile(ctx, cclfFileSuccess)
	assert.NoError(err)
	cclfFileSuccessOld.ID, err = r.repository.CreateCCLFFile(ctx, cclfFileSuccessOld)
	assert.NoError(err)
	cclfFileOther.ID, err = r.repository.CreateCCLFFile(ctx, cclfFileOther)
	assert.NoError(err)

	cclfFile, err := r.repository.GetLatestCCLFFile(ctx, cclfFileFailed.ACOCMSID, cclfFileFailed.CCLFNum, cclfFileFailed.ImportStatus,
		time.Time{}, time.Time{}, cclfFileFailed.Type)
	assert.NoError(err)
	assertEqualCCLFFile(assert, cclfFileFailed, *cclfFile)

	cclfFile, err = r.repository.GetLatestCCLFFile(ctx, cclfFileSuccess.ACOCMSID, cclfFileSuccess.CCLFNum, cclfFileSuccess.ImportStatus,
		time.Time{}, time.Time{}, cclfFileFailed.Type)
	assert.NoError(err)
	// expect cclfFileSuccess since it's newer than cclfFileSuccessOld
	assertEqualCCLFFile(assert, cclfFileSuccess, *cclfFile)

	cclfFile, err = r.repository.GetLatestCCLFFile(ctx, cclfFileOther.ACOCMSID, cclfFileOther.CCLFNum, cclfFileOther.ImportStatus,
		time.Time{}, time.Time{}, cclfFileOther.Type)
	assert.NoError(err)
	assertEqualCCLFFile(assert, cclfFileOther, *cclfFile)

	cclfFileOther.ImportStatus = "Other2"
	assert.NoError(r.repository.UpdateCCLFFileImportStatus(ctx, cclfFileOther.ID, cclfFileOther.ImportStatus))
	assertEqualCCLFFile(assert, cclfFileOther, postgrestest.GetCCLFFilesByName(r.T(), r.db, cclfFileOther.Name)[0])
	assertEqualCCLFFile(assert, cclfFileFailed, postgrestest.GetCCLFFilesByName(r.T(), r.db, cclfFileFailed.Name)[0])
	assertEqualCCLFFile(assert, cclfFileSuccess, postgrestest.GetCCLFFilesByName(r.T(), r.db, cclfFileSuccess.Name)[0])
	assertEqualCCLFFile(assert, cclfFileSuccessOld, postgrestest.GetCCLFFilesByName(r.T(), r.db, cclfFileSuccessOld.Name)[0])

	// Negative tests
	_, err = r.repository.CreateCCLFFile(ctx, cclfFileSuccess)
	assert.Contains(err.Error(), "duplicate key value violates unique constraint \"idx_cclf_files_name_aco_cms_id_key\"")
	assert.EqualError(r.repository.UpdateCCLFFileImportStatus(ctx, 0, "Other3"), "failed to update file entry 0 status to Other3, no entry found")
	_, err = r.repository.GetLatestCCLFFile(ctx, testUtils.RandomHexID(), -1, "", time.Time{}, time.Time{}, models.FileTypeDefault)
	assert.NoError(err)
}

// TestCCLFBeneficiariesMethods validates the CRUD operations associated with the cclf_beneficiaries table
func (r *RepositoryTestSuite) TestCCLFBeneficiariesMethods() {
	ctx := context.Background()
	assert := r.Assert()

	// Since we have a foreign key tie, we need the cclf file to exist before creating associated benes
	cclfFile := &models.CCLFFile{CCLFNum: 8, ACOCMSID: testUtils.RandomHexID()[0:4], Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	postgrestest.CreateCCLFFile(r.T(), r.db, cclfFile)
	defer postgrestest.DeleteCCLFFilesByCMSID(r.T(), r.db, cclfFile.ACOCMSID)

	bene1 := &models.CCLFBeneficiary{FileID: cclfFile.ID, MBI: testUtils.RandomMBI(r.T()), BlueButtonID: testUtils.RandomHexID()}
	bene2 := &models.CCLFBeneficiary{FileID: cclfFile.ID, MBI: testUtils.RandomMBI(r.T()), BlueButtonID: testUtils.RandomHexID()}
	postgrestest.CreateCCLFBeneficiary(r.T(), r.db, bene1)
	postgrestest.CreateCCLFBeneficiary(r.T(), r.db, bene2)

	mbis, err := r.repository.GetCCLFBeneficiaryMBIs(ctx, cclfFile.ID)
	assert.NoError(err)
	assert.Len(mbis, 2)
	assert.Contains(mbis, bene1.MBI)
	assert.Contains(mbis, bene2.MBI)

	benes, err := r.repository.GetCCLFBeneficiaries(ctx, cclfFile.ID, nil)
	assert.NoError(err)
	assert.Len(benes, 2)
	assert.Contains(benes, bene1)
	assert.Contains(benes, bene2)

	// All benes excluded
	benes, err = r.repository.GetCCLFBeneficiaries(ctx, cclfFile.ID, mbis)
	assert.NoError(err)
	assert.Len(benes, 0)

	// Negative cases
	mbis, err = r.repository.GetCCLFBeneficiaryMBIs(ctx, 0)
	assert.NoError(err)
	assert.Len(mbis, 0)

	benes, err = r.repository.GetCCLFBeneficiaries(ctx, 0, mbis)
	assert.NoError(err)
	assert.Len(benes, 0)
}

// TestSuppressionsMethods validates the CRUD operations associated with the suppressions table
func (r *RepositoryTestSuite) TestSuppresionsMethods() {
	ctx := context.Background()
	assert := r.Assert()
	fileID := uint(rand.Int31())
	upperBound := time.Now().Add(-30 * time.Minute)
	// Effective date is too old
	tooOld := models.Suppression{FileID: fileID, MBI: testUtils.RandomMBI(r.T()), PrefIndicator: "N",
		EffectiveDt: time.Now().Add(-365 * 24 * time.Hour)}
	// Effective date is after the upper bound
	tooNew := models.Suppression{FileID: fileID, MBI: testUtils.RandomMBI(r.T()), PrefIndicator: "N",
		EffectiveDt: time.Now()}
	// Mismatching preference indicators
	mismatch1 := models.Suppression{FileID: fileID, MBI: testUtils.RandomMBI(r.T()), PrefIndicator: "Y",
		EffectiveDt: time.Now().Add(-time.Hour)}
	mismatch2 := models.Suppression{FileID: fileID, MBI: testUtils.RandomMBI(r.T()), PrefIndicator: "",
		EffectiveDt: time.Now().Add(-time.Hour)}
	suppressed1 := models.Suppression{FileID: fileID, MBI: testUtils.RandomMBI(r.T()), PrefIndicator: "N",
		EffectiveDt: time.Now().Add(-time.Hour)}
	suppressed2 := models.Suppression{FileID: fileID, MBI: testUtils.RandomMBI(r.T()), PrefIndicator: "N",
		EffectiveDt: time.Now().Add(-time.Hour)}

	assert.NoError(r.repository.CreateSuppression(ctx, tooOld))
	assert.NoError(r.repository.CreateSuppression(ctx, tooNew))
	assert.NoError(r.repository.CreateSuppression(ctx, mismatch1))
	assert.NoError(r.repository.CreateSuppression(ctx, mismatch2))
	assert.NoError(r.repository.CreateSuppression(ctx, suppressed1))
	assert.NoError(r.repository.CreateSuppression(ctx, suppressed2))

	defer postgrestest.DeleteSuppressionFileByID(r.T(), r.db, fileID)

	mbis, err := r.repository.GetSuppressedMBIs(ctx, 10, upperBound)
	assert.NoError(err)

	// Since we have other data seeded in this table, we cannot do a len check on the MBIs
	assert.Contains(mbis, suppressed1.MBI)
	assert.Contains(mbis, suppressed2.MBI)
	assert.NotContains(mbis, tooOld.MBI)
	assert.NotContains(mbis, tooNew.MBI)
	assert.NotContains(mbis, mismatch1.MBI)
	assert.NotContains(mbis, mismatch2.MBI)
}

// TestSuppressionFilesMethods validates the CRUD operations associated with the suppression_files table
func (r *RepositoryTestSuite) TestSuppressionFileMethods() {
	// Account for time precision in postgres
	now := time.Now().Round(time.Millisecond)
	var err error
	ctx := context.Background()
	assert := r.Assert()

	inProgress := models.SuppressionFile{
		Name:         uuid.New(),
		Timestamp:    now,
		ImportStatus: constants.ImportInprog,
	}
	failed := models.SuppressionFile{
		Name:         uuid.New(),
		Timestamp:    now,
		ImportStatus: constants.ImportFail,
	}
	other := models.SuppressionFile{
		Name:         uuid.New(),
		Timestamp:    now,
		ImportStatus: "Other",
	}

	inProgress.ID, err = r.repository.CreateSuppressionFile(ctx, inProgress)
	assert.NoError(err)
	failed.ID, err = r.repository.CreateSuppressionFile(ctx, failed)
	assert.NoError(err)
	other.ID, err = r.repository.CreateSuppressionFile(ctx, other)
	assert.NoError(err)

	inProgress.ImportStatus = "Completed"
	assert.NoError(r.repository.UpdateSuppressionFileImportStatus(ctx, inProgress.ID, inProgress.ImportStatus))

	assertEqualSuppressionFile(assert, inProgress, postgrestest.GetSuppressionFileByName(r.T(), r.db, inProgress.Name)[0])
	assertEqualSuppressionFile(assert, failed, postgrestest.GetSuppressionFileByName(r.T(), r.db, failed.Name)[0])
	assertEqualSuppressionFile(assert, other, postgrestest.GetSuppressionFileByName(r.T(), r.db, other.Name)[0])

	// Negative cases
	assert.EqualError(r.repository.UpdateSuppressionFileImportStatus(ctx, 0, "SomeOtherStatus"), "SuppressionFile 0 not updated, no row found")
}

// TestJobsMethods validates the CRUD operations associated with the jobs table
func (r *RepositoryTestSuite) TestJobsMethods() {
	var err error
	ctx := context.Background()
	assert := r.Assert()

	reqURL := "http://bcda.cms.gov/is/the/best"

	cmsID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{UUID: uuid.NewRandom(), Name: uuid.New(), CMSID: &cmsID}
	postgrestest.CreateACO(r.T(), r.db, aco)

	defer postgrestest.DeleteACO(r.T(), r.db, aco.UUID)

	failed := models.Job{ACOID: aco.UUID, RequestURL: reqURL, Status: models.JobStatusFailed, JobCount: 10, CompletedJobCount: 20}
	pending := models.Job{ACOID: aco.UUID, RequestURL: reqURL, Status: models.JobStatusPending, JobCount: 30, CompletedJobCount: 40}
	completed := models.Job{ACOID: aco.UUID, RequestURL: reqURL, Status: models.JobStatusCompleted, JobCount: 40, CompletedJobCount: 60}

	failed.ID, err = r.repository.CreateJob(ctx, failed)
	assert.NoError(err)
	pending.ID, err = r.repository.CreateJob(ctx, pending)
	assert.NoError(err)
	completed.ID, err = r.repository.CreateJob(ctx, completed)
	assert.NoError(err)

	jobs, err := r.repository.GetJobs(ctx, aco.UUID)

	// Used to track certain range queries
	var earliestTime, latestTime time.Time

	// Since updatedAt and createdAt are computed fields e.g. not set on the model, we'll make sure
	// postgres set it as expected
	for _, j := range jobs {
		assert.False(j.CreatedAt.IsZero())
		assert.False(j.UpdatedAt.IsZero())

		if earliestTime.IsZero() || !earliestTime.Before(j.UpdatedAt) {
			earliestTime = j.UpdatedAt
		}
		if latestTime.IsZero() || !latestTime.After(j.UpdatedAt) {
			latestTime = j.UpdatedAt
		}
	}

	assert.NoError(err)
	assert.Len(jobs, 3)
	assertContainsJobID(assert, jobs, failed.ID)
	assertContainsJobID(assert, jobs, completed.ID)
	assertContainsJobID(assert, jobs, pending.ID)

	jobs, err = r.repository.GetJobs(ctx, aco.UUID, models.JobStatusFailed)
	assert.NoError(err)
	assert.Len(jobs, 1)
	assertContainsJobID(assert, jobs, failed.ID)

	// Since other jobs could've been created and we don't limit by UUID
	// we can't guarantee counts
	jobs, err = r.repository.GetJobsByUpdateTimeAndStatus(ctx, earliestTime, latestTime)
	assert.NoError(err)
	assertContainsJobID(assert, jobs, failed.ID)
	assertContainsJobID(assert, jobs, completed.ID)
	assertContainsJobID(assert, jobs, pending.ID)

	jobs, err = r.repository.GetJobsByUpdateTimeAndStatus(ctx, earliestTime, latestTime, models.JobStatusFailed)
	assert.NoError(err)
	assertContainsJobID(assert, jobs, failed.ID)
	assertDoesNotContainsJobID(assert, jobs, completed.ID)
	assertDoesNotContainsJobID(assert, jobs, pending.ID)

	// All jobs were created before this timebound.
	jobs, err = r.repository.GetJobsByUpdateTimeAndStatus(ctx, latestTime.Add(1*time.Minute), time.Time{})
	assert.NoError(err)
	assertDoesNotContainsJobID(assert, jobs, failed.ID)
	assertDoesNotContainsJobID(assert, jobs, completed.ID)
	assertDoesNotContainsJobID(assert, jobs, pending.ID)

	// All jobs were created after this timebound.
	jobs, err = r.repository.GetJobsByUpdateTimeAndStatus(ctx, time.Time{}, earliestTime.Add(-1*time.Minute))
	assert.NoError(err)
	assertDoesNotContainsJobID(assert, jobs, failed.ID)
	assertDoesNotContainsJobID(assert, jobs, completed.ID)
	assertDoesNotContainsJobID(assert, jobs, pending.ID)

	// Account for time precision in postgres
	failed.TransactionTime = time.Now().Round(time.Millisecond)
	failed.JobCount = failed.JobCount + 1
	failed.CompletedJobCount = failed.CompletedJobCount + 1
	failed.Status = models.JobStatusArchived
	assert.NoError(r.repository.UpdateJob(ctx, failed))

	newFailed, err := r.repository.GetJobByID(ctx, failed.ID)
	assert.NoError(err)
	assert.True(newFailed.UpdatedAt.After(failed.UpdatedAt))
	assert.Equal(failed.TransactionTime.UTC(), newFailed.TransactionTime.UTC())
	assert.Equal(failed.Status, newFailed.Status)
	assert.Equal(failed.JobCount, newFailed.JobCount)
	assert.Equal(failed.CompletedJobCount, newFailed.CompletedJobCount)

	// Verify that we did not modify other job
	newCompleted, err := r.repository.GetJobByID(ctx, completed.ID)
	assert.NoError(err)
	assert.Equal(models.JobStatusCompleted, newCompleted.Status)
	assert.True(newFailed.UpdatedAt.After(newCompleted.UpdatedAt))

	// Negative cases
	notExists := models.Job{ACOID: aco.UUID, RequestURL: reqURL, Status: models.JobStatusCompleted}
	assert.EqualError(r.repository.UpdateJob(ctx, notExists), "expected to affect 1 row, affected 0")
}

// TestJobKeysMethods validates the CRUD operations associated with the job_keys table
func (r *RepositoryTestSuite) TestJobKeyMethods() {
	ctx := context.Background()
	assert := r.Assert()

	jobID := uint(rand.Int31())
	jk1 := models.JobKey{JobID: jobID, FileName: uuid.New()}
	jk2 := models.JobKey{JobID: jobID, FileName: uuid.New()}
	jk3 := models.JobKey{JobID: uint(rand.Int31()), FileName: uuid.New()}

	postgrestest.CreateJobKeys(r.T(), r.db, jk1, jk2, jk3)

	// Since we have other job keys that exist, we cannot guarantee length
	keys, err := r.repository.GetJobKeys(ctx, jobID)
	assert.NoError(err)
	assertContainsFile(assert, keys, jk1.FileName)
	assertContainsFile(assert, keys, jk2.FileName)
	assertDoesNotContainsFile(assert, keys, jk3.FileName)

	otherKeys, err := r.repository.GetJobKeys(ctx, jk3.JobID)
	assert.NoError(err)
	assertContainsFile(assert, otherKeys, jk3.FileName)
	assertDoesNotContainsFile(assert, otherKeys, jk1.FileName)
	assertDoesNotContainsFile(assert, otherKeys, jk2.FileName)
}

// TestCMSID verifies that we can store and retrieve the CMS_ID as expected
// i.e. the value is not padded with any extra characters
func (r *RepositoryTestSuite) TestCMSID() {
	cmsID := "V001"
	cclfFile := models.CCLFFile{CCLFNum: 1, Name: "someName", ACOCMSID: cmsID, Timestamp: time.Now(), PerformanceYear: 20}
	aco := models.ACO{UUID: uuid.NewUUID(), CMSID: &cmsID, Name: "someName"}
	var err error

	cclfFile.ID, err = r.repository.CreateCCLFFile(context.Background(), cclfFile)
	assert.NoError(r.T(), err)
	defer postgrestest.DeleteCCLFFilesByCMSID(r.T(), r.db, cmsID)

	postgrestest.CreateACO(r.T(), r.db, aco)
	defer postgrestest.DeleteACO(r.T(), r.db, aco.UUID)

	actualCMSID := *postgrestest.GetACOByUUID(r.T(), r.db, aco.UUID).CMSID
	assert.Equal(r.T(), cmsID, actualCMSID)

	actualCMSID = postgrestest.GetCCLFFilesByName(r.T(), r.db, cclfFile.Name)[0].ACOCMSID
	assert.Equal(r.T(), cmsID, actualCMSID)
}

func (r *RepositoryTestSuite) TestCCLFFileType() {
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

	defer postgrestest.DeleteCCLFFilesByCMSID(r.T(), r.db, cmsID)
	noType.ID, err = r.repository.CreateCCLFFile(context.Background(), noType)
	assert.NoError(r.T(), err)

	withType.ID, err = r.repository.CreateCCLFFile(context.Background(), withType)
	assert.NoError(r.T(), err)

	result := postgrestest.GetCCLFFilesByName(r.T(), r.db, noType.Name)
	assert.Equal(r.T(), 1, len(result))
	assert.Equal(r.T(), noType.Type, result[0].Type)

	result = postgrestest.GetCCLFFilesByName(r.T(), r.db, withType.Name)
	assert.Equal(r.T(), 1, len(result))
	assert.Equal(r.T(), withType.Type, result[0].Type)
}

func getCCLFFile(cclfNum int, cmsID, importStatus string, fileType models.CCLFFileType) *models.CCLFFile {
	// Account for time precision in postgres
	createTime := time.Now().Round(time.Millisecond)
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

func assertEqualCCLFFile(assert *assert.Assertions, expected, actual models.CCLFFile) {
	// normalize timestamps so we can use equality checks
	expected.Timestamp = expected.Timestamp.UTC()
	actual.Timestamp = actual.Timestamp.UTC()

	assert.Equal(expected, actual)
}

func assertEqualSuppressionFile(assert *assert.Assertions, expected, actual models.SuppressionFile) {
	// normalize timestamps so we can use equality checks
	expected.Timestamp = expected.Timestamp.UTC()
	actual.Timestamp = actual.Timestamp.UTC()

	assert.Equal(expected, actual)
}

func assertContainsJobID(assert *assert.Assertions, jobs []*models.Job, jobID uint) {
	var jobIDs []uint
	for _, job := range jobs {
		jobIDs = append(jobIDs, job.ID)
	}

	assert.Contains(jobIDs, jobID)
}

func assertDoesNotContainsJobID(assert *assert.Assertions, jobs []*models.Job, jobID uint) {
	jobIDs := make(map[uint]struct{})
	for _, job := range jobs {
		jobIDs[job.ID] = struct{}{}
	}

	_, contains := jobIDs[jobID]
	assert.False(contains, "JobIDs %v should not include %d", jobIDs, jobID)
}

func assertContainsFile(assert *assert.Assertions, jobKeys []*models.JobKey, fileName string) {
	var fileNames []string
	for _, jobKey := range jobKeys {
		// need to use trimSpace because the jobKey#fileName can be padded with extra characters
		fileNames = append(fileNames, strings.TrimSpace(jobKey.FileName))
	}

	assert.Contains(fileNames, fileName)
}

func assertDoesNotContainsFile(assert *assert.Assertions, jobKeys []*models.JobKey, fileName string) {
	fileNames := make(map[string]struct{})
	for _, jobKey := range jobKeys {
		// need to use trimSpace because the jobKey#fileName can be padded with extra characters
		fileNames[strings.TrimSpace(jobKey.FileName)] = struct{}{}
	}

	_, contains := fileNames[fileName]
	assert.False(contains, "File names %v should not include %d", fileNames, fileName)
}
