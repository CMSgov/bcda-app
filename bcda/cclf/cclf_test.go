package cclf

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"

	"github.com/CMSgov/bcda-app/bcda/testUtils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type CCLFTestSuite struct {
	suite.Suite
	pendingDeletionDir string

	basePath string
	cleanup  func()

	origDate string

	db *sql.DB
}

func (s *CCLFTestSuite) SetupTest() {
	os.Setenv("CCLF_REF_DATE", "181201")

	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
}

func (s *CCLFTestSuite) SetupSuite() {
	s.origDate = os.Getenv("CCLF_REF_DATE")

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)

	s.db = database.GetDbConnection()
}

func (s *CCLFTestSuite) TearDownSuite() {
	os.Setenv("CCLF_REF_DATE", s.origDate)
	os.RemoveAll(s.pendingDeletionDir)
	s.db.Close()
}

func (s *CCLFTestSuite) TearDownTest() {
	s.cleanup()
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFTestSuite))
}

func (s *CCLFTestSuite) TestImportCCLFDirectory_PriorityACOs() {
	// The order they should be ingested in. 1 and 2 are prioritized; 3 is the other ACO in the directory.
	// This order is computed from values inserted in the database
	var aco1, aco2, aco3 = "A9989", "A9988", "A0001"

	os.Setenv("CCLF_REF_DATE", "181201")

	assert := assert.New(s.T())

	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, aco1)
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, aco2)
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, aco3)

	sc, f, sk, err := ImportCCLFDirectory(filepath.Join(s.basePath, "cclf/archives/valid/"))
	assert.Nil(err)
	assert.Equal(6, sc)
	assert.Equal(0, f)
	assert.Equal(1, sk)

	aco1fs := postgrestest.GetCCLFFilesByCMSID(s.T(), s.db, aco1)
	aco2fs := postgrestest.GetCCLFFilesByCMSID(s.T(), s.db, aco2)
	aco3fs := postgrestest.GetCCLFFilesByCMSID(s.T(), s.db, aco3)

	assert.True(aco1fs[0].ID < aco2fs[0].ID)
	assert.True(aco2fs[0].ID < aco3fs[0].ID)
}

func (s *CCLFTestSuite) TestImportCCLF0() {
	ctx := context.Background()
	assert := assert.New(s.T())

	cclf0filePath := filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000")
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011"}

	// positive
	validator, err := importCCLF0(ctx, cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 7, maxRecordLength: 549}, validator["CCLF8"])

	// negative
	cclf0metadata = &cclfFileMetadata{}
	_, err = importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "could not read CCLF0 archive : read .: is a directory")

	// missing cclf8 from cclf0
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181120.T1000000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011"}
	_, err = importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "failed to parse CCLF8 from CCLF0 file T.BCD.A0001.ZC0Y18.D181120.T1000011")

	// duplicate file types from cclf0
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181122.T1000000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000013"}
	_, err = importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "duplicate CCLF8 file type found from CCLF0 file")
}

func (s *CCLFTestSuite) TestImportCCLF0_SplitFiles() {
	assert := assert.New(s.T())

	cclf0filePath := filepath.Join(s.basePath, "cclf/archives/split/T.BCD.A0001.ZCY18.D181120.T1000000")
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011-1"}

	validator, err := importCCLF0(context.Background(), cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])
}

func (s *CCLFTestSuite) TestValidate() {
	ctx := context.Background()
	assert := assert.New(s.T())

	cclf8filePath := filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000")
	cclf8metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18, name: "T.BCD.A0001.ZC8Y18.D181120.T1000009"}

	// positive
	cclfvalidator := map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 7, maxRecordLength: 549}}
	err := validate(ctx, cclf8metadata, cclfvalidator)
	assert.Nil(err)

	// negative
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 2, maxRecordLength: 549}}
	err = validate(ctx, cclf8metadata, cclfvalidator)
	assert.EqualError(err, "maximum record count reached for file CCLF8 (expected: 2, actual: 3)")
}

func (s *CCLFTestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())

	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &cclfFileMetadata{
		name:      "T.BCD.A0001.ZC8Y18.D181120.T1000009",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000"),
	}

	err := importCCLF8(context.Background(), metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := postgrestest.GetCCLFFilesByName(s.T(), s.db, metadata.name)[0]
	assert.Equal("T.BCD.A0001.ZC8Y18.D181120.T1000009", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime.Format("010203040506"), file.Timestamp.Format("010203040506"))
	assert.Equal(18, file.PerformanceYear)
	assert.Equal(constants.ImportComplete, file.ImportStatus)

	mbis, err := postgres.NewRepository(s.db).GetCCLFBeneficiaryMBIs(context.Background(), file.ID)
	assert.NoError(err)

	assert.Len(mbis, 6)
	sort.Strings(mbis)
	assert.Equal("1A69B98CD30", mbis[0])
	assert.Equal("1A69B98CD31", mbis[1])
	assert.Equal("1A69B98CD32", mbis[2])
	assert.Equal("1A69B98CD33", mbis[3])
	assert.Equal("1A69B98CD34", mbis[4])
	assert.Equal("1A69B98CD35", mbis[5])
}

func (s *CCLFTestSuite) TestImportCCLF8_Invalid() {
	assert := assert.New(s.T())

	var metadata *cclfFileMetadata
	err := importCCLF8(context.Background(), metadata)
	assert.EqualError(err, "CCLF file not found")

	// since we do not have the correct number of characters, the import should fail.
	fileName, cclfName := createTemporaryCCLF8ZipFile(s.T(), "A 1")
	defer os.Remove(fileName)
	metadata = &cclfFileMetadata{
		cclfNum:   8,
		name:      cclfName,
		acoID:     testUtils.RandomHexID()[0:4],
		timestamp: time.Now(),
		perfYear:  20,
		filePath:  fileName,
	}
	err = importCCLF8(context.Background(), metadata)
	// This error indicates that we did not supply enough characters for the MBI
	assert.Contains(err.Error(), "invalid byte sequence for encoding \"UTF8\": 0x00")
}

func (s *CCLFTestSuite) TestOrderACOs() {
	var cclfMap = map[string]map[metadataKey][]*cclfFileMetadata{
		"A1111": map[metadataKey][]*cclfFileMetadata{},
		"A8765": map[metadataKey][]*cclfFileMetadata{},
		"A3456": map[metadataKey][]*cclfFileMetadata{},
		"A0246": map[metadataKey][]*cclfFileMetadata{},
	}

	acoOrder := orderACOs(cclfMap)

	// A3456 and A8765 have been added to the database == prioritized over the other two
	assert.Len(s.T(), acoOrder, 4)
	assert.Equal(s.T(), "A3456", acoOrder[0])
	assert.Equal(s.T(), "A8765", acoOrder[1])
	assert.Regexp(s.T(), "A1111|A0246", acoOrder[2])
	assert.Regexp(s.T(), "A1111|A0246", acoOrder[3])
}

func (s *CCLFTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[metadataKey][]*cclfFileMetadata)

	// failed import: file that's within the threshold - stay put
	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf0metadata := &cclfFileMetadata{
		name:         "T.BCD.ACO.ZC0Y18.D181120.T0001000",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "cclf/T.BCD.ACO.ZC0Y18.D181120.T0001000"),
		imported:     false,
		deliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf8metadata := &cclfFileMetadata{
		name:         "T.BCD.A0001.ZC8Y18.D181120.T1000009",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000"),
		imported:     false,
		deliveryDate: fileTime,
	}

	// successfully imported file - should move
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf9metadata := &cclfFileMetadata{
		name:      "T.BCD.A0001.ZC9Y18.D181120.T1000010",
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181122.T1000000"),
		imported:  true,
	}
	cclfmap["A0001"] = map[metadataKey][]*cclfFileMetadata{
		{perfYear: 18, fileType: models.FileTypeDefault}: {cclf0metadata, cclf8metadata, cclf9metadata},
	}
	err := cleanUpCCLF(context.Background(), cclfmap)
	assert.Nil(err)

	files, err := ioutil.ReadDir(os.Getenv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", os.Getenv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual("T.BCD.ACO.ZC0Y18.D181120.T0001000", file.Name())
	}
}

func (s *CCLFTestSuite) TestGetPriorityACOs() {
	query := regexp.QuoteMeta(`
	SELECT trim(both '["]' from g.x_data::json->>'cms_ids') "aco_id" 
	FROM systems s JOIN groups g ON s.group_id=g.group_id 
	WHERE s.deleted_at IS NULL AND g.group_id IN (SELECT group_id FROM groups WHERE x_data LIKE '%A%' and x_data NOT LIKE '%A999%') AND
	s.id IN (SELECT system_id FROM secrets WHERE deleted_at IS NULL);
	`)
	tests := []struct {
		name        string
		idsToReturn []string
		errToReturn error
	}{
		{"ErrorOnQuery", nil, errors.New("Some query error")},
		{"NoActiveACOs", nil, nil},
		{"ActiveACOs", []string{"A", "B", "C", "123"}, nil},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			}()

			expected := mock.ExpectQuery(query)
			if tt.errToReturn != nil {
				expected.WillReturnError(tt.errToReturn)
			} else {
				rows := sqlmock.NewRows([]string{"cms_id"})
				for _, id := range tt.idsToReturn {
					rows.AddRow(id)
				}
				expected.WillReturnRows(rows)
			}

			result := getPriorityACOs(db)
			if tt.errToReturn != nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.idsToReturn, result)
			}
		})
	}
}

func (s *CCLFTestSuite) TestImportRunoutCCLF() {
	db := database.GetDbConnection()
	defer db.Close()

	cmsID := "RNOUT"
	defer func() {
		postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, cmsID)
	}()

	tests := []struct {
		name     string
		fileType models.CCLFFileType
	}{
		{"Default file type", models.FileTypeDefault},
		{"Runout file type", models.FileTypeRunout},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mbi := "123456789AB" // We expect 11 characters for the MBI

			fileName, cclfName := createTemporaryCCLF8ZipFile(s.T(), mbi)
			defer os.Remove(fileName)

			metadata := &cclfFileMetadata{
				cclfNum:   8,
				name:      cclfName,
				acoID:     cmsID,
				timestamp: time.Now(),
				perfYear:  20,
				fileType:  tt.fileType,
				filePath:  fileName,
			}

			s.NoError(importCCLF8(context.Background(), metadata))

			cclfFile := postgrestest.GetCCLFFilesByName(s.T(), db, cclfName)[0]
			assert.Equal(t, tt.fileType, cclfFile.Type)
		})
	}
}

func createTemporaryCCLF8ZipFile(t *testing.T, data string) (fileName, cclfName string) {
	cclfName = uuid.New()

	f, err := ioutil.TempFile("", "*")
	assert.NoError(t, err)

	w := zip.NewWriter(f)
	f1, err := w.Create(cclfName)
	assert.NoError(t, err)

	_, err = f1.Write([]byte(data))
	assert.NoError(t, err)

	assert.NoError(t, w.Close())

	return f.Name(), cclfName
}
