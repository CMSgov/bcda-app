package cclf

import (
	"archive/zip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CCLFTestSuite struct {
	suite.Suite
	pendingDeletionDir string

	basePath string
	importer CclfImporter
	cleanup  func()

	origDate string

	db *sql.DB
}

func (s *CCLFTestSuite) SetupTest() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201")

	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")

	file_handler := &optout.LocalFileHandler{
		Logger:                 log.API,
		PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
		FileArchiveThresholdHr: uint(utils.GetEnvInt("FILE_ARCHIVE_THRESHOLD_HR", 72)),
	}

	file_processor := &LocalFileProcessor{}
	s.importer = CclfImporter{
		FileHandler:   file_handler,
		FileProcessor: file_processor,
	}
}

func (s *CCLFTestSuite) SetupSuite() {
	s.origDate = conf.GetEnv("CCLF_REF_DATE")

	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)

	s.db = database.Connection
}

func (s *CCLFTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.origDate)
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *CCLFTestSuite) TearDownTest() {
	s.cleanup()
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFTestSuite))
}

func (s *CCLFTestSuite) TestImportCCLF0() {
	ctx := context.Background()
	assert := assert.New(s.T())

	cclf0filePath := filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000")
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011"}

	// Missing metadata
	_, err := s.importer.importCCLF0(ctx, nil)
	assert.EqualError(err, "file CCLF0 not found")

	// positive
	validator, err := s.importer.importCCLF0(ctx, cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 7, maxRecordLength: 549}, validator["CCLF8"])

	// negative
	cclf0metadata = &cclfFileMetadata{}
	_, err = s.importer.importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "could not read CCLF0 archive : open : no such file or directory")

	// missing cclf8 from cclf0
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181120.T1000000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011"}
	_, err = s.importer.importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "failed to parse CCLF8 from CCLF0 file T.BCD.A0001.ZC0Y18.D181120.T1000011")

	// duplicate file types from cclf0
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181122.T1000000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000013"}
	_, err = s.importer.importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "duplicate CCLF8 file type found from CCLF0 file")

	// missing file
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181122.T1000000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "Z.BCD.A0001.ZC0Y18.D181120.T1000013"}
	_, err = s.importer.importCCLF0(ctx, cclf0metadata)
	assert.Nil(err)

	//invalid record count
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/invalid/T.A0001.ACO.ZC0Y18.D181120.Z1000000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.A0001.ACO.ZC0Y18.D181120.Z1000011"}
	_, err = s.importer.importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "failed to parse CCLF8 record count from CCLF0 file: strconv.Atoi: parsing \"N\": invalid syntax")

	//invalid record length
	cclf0filePath = filepath.Join(s.basePath, "cclf/archives/0/invalid/T.BCD.ACOB.ZC0Y18.D181120.E0001000")
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.A0001.ACO.ZC0Y18.D181120.E1000011"}
	_, err = s.importer.importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "failed to parse CCLF8 record length from CCLF0 file: strconv.Atoi: parsing \"Num\": invalid syntax")

}

func (s *CCLFTestSuite) TestImportCCLF0_SplitFiles() {
	assert := assert.New(s.T())

	cclf0filePath := filepath.Join(s.basePath, "cclf/archives/split/T.BCD.A0001.ZCY18.D181120.T1000000")
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011-1"}

	validator, err := s.importer.importCCLF0(context.Background(), cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])
}

func (s *CCLFTestSuite) TestImportCCLFDirectoryValid() {
	assert := assert.New(s.T())
	//Happy case, with directory containing valid BCD files.
	_, _, _, err := s.importer.ImportCCLFDirectory(filepath.Join(s.basePath, constants.CCLFDIR, "archives", "valid_bcd"))
	assert.Nil(err)
}
func (s *CCLFTestSuite) TestImportCCLFDirectoryInvalid() {
	assert := assert.New(s.T())
	//Directory with mixed file types + at least one bad file.
	cclfDirectory := filepath.Join(s.basePath, constants.CCLFDIR)
	_, _, _, err := s.importer.ImportCCLFDirectory(cclfDirectory)
	assert.EqualError(err, "one or more files failed to import correctly")

	//Target bad file directory
	cclfDirectory = filepath.Join(s.basePath, constants.CCLFDIR, "archives", "invalid_bcd")
	_, _, _, err = s.importer.ImportCCLFDirectory(cclfDirectory)
	assert.EqualError(err, "one or more files failed to import correctly")

}

/*
	func (s *CCLFTestSuite) TestImportCCLFDirectoryEmpty() {
		assert := assert.New(s.T())
		//Zero CCLF files in directory
		cclfDirectory := filepath.Join(s.basePath, constants.CCLFDIR, "emptydir")
		_, _, _, err := ImportCCLFDirectory(cclfDirectory)
		assert.Nil(err)
	}
*/
func (s *CCLFTestSuite) TestImportCCLFDirectoryTwoLevels() {
	assert := assert.New(s.T())
	//Zero CCLF files in directory
	//additional invalid directory
	cclfDirectory := filepath.Join(s.basePath, constants.CCLFDIR, "emptydir", "archives")
	_, _, _, err := s.importer.ImportCCLFDirectory(cclfDirectory)
	assert.EqualError(err, "error in sorting cclf file: nil,: lstat "+cclfDirectory+": no such file or directory")

}

func (s *CCLFTestSuite) TestValidate() {
	ctx := context.Background()
	assert := assert.New(s.T())

	cclf8filePath := filepath.Join(s.basePath, constants.CCLF8CompPath)
	cclf8metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18, name: constants.CCLF8Name}

	// missing metadata
	cclfvalidator := map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 7, maxRecordLength: 549}}
	err := s.importer.validate(ctx, nil, cclfvalidator)
	assert.EqualError(err, "file not found")

	// positive
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 7, maxRecordLength: 549}}
	err = s.importer.validate(ctx, cclf8metadata, cclfvalidator)
	assert.Nil(err)

	// negative
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 2, maxRecordLength: 549}}
	err = s.importer.validate(ctx, cclf8metadata, cclfvalidator)
	assert.EqualError(err, "maximum record count reached for file CCLF8 (expected: 2, actual: 3)")

	//invalid cclfNum
	cclf8metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 9, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18, name: constants.CCLF8Name}
	err = s.importer.validate(ctx, cclf8metadata, cclfvalidator)
	assert.EqualError(err, "unknown file type when validating file: T.BCD.A0001.ZC8Y18.D181120.T1000009")

	//invalid file path
	cclf8metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: "/", perfYear: 18, name: constants.CCLF8Name}
	err = s.importer.validate(ctx, cclf8metadata, cclfvalidator)
	assert.EqualError(err, "could not read archive /: read /: is a directory")

	//non-existant file
	cclf8metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18, name: "InvalidName"}
	err = s.importer.validate(ctx, cclf8metadata, cclfvalidator)
	assert.Nil(err)

	//more records than expected
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 7, maxRecordLength: 0}}
	cclf8metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18, name: constants.CCLF8Name}
	err = s.importer.validate(ctx, cclf8metadata, cclfvalidator)
	assert.EqualError(err, "incorrect record length for file CCLF8 (expected: 0, actual: 549)")

}

func (s *CCLFTestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())

	//indeterminate test results without deletion of both.
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0002")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0002")

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, constants.TestFileTime)

	//invalid directory
	metadata := &cclfFileMetadata{
		name:      constants.CCLF8Name,
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  "/missingdir",
	}
	err := s.importer.importCCLF8(context.Background(), metadata)
	assert.EqualError(err, "could not read CCLF8 archive /missingdir: open /missingdir: no such file or directory")

	//No files in CCLF zip
	metadata = &cclfFileMetadata{
		name:      constants.CCLF8Name,
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, "cclf/archives/8/invalid/T.BCD.A0001.ZCY18.D181125.T1000087"),
	}
	err = s.importer.importCCLF8(context.Background(), metadata)
	assert.EqualError(err, "no files found in CCLF8 archive "+metadata.filePath)

	//file not found
	metadata = &cclfFileMetadata{
		name:      constants.CCLF8Name + "E",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, constants.CCLF8CompPath),
	}

	err = s.importer.importCCLF8(context.Background(), metadata)
	assert.EqualError(err, "file "+metadata.name+" not found in archive "+metadata.filePath)

	//successful
	metadata = &cclfFileMetadata{
		name:      constants.CCLF8Name,
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, constants.CCLF8CompPath),
	}

	err = s.importer.importCCLF8(context.Background(), metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := postgrestest.GetCCLFFilesByName(s.T(), s.db, metadata.name)[0]
	assert.Equal(constants.CCLF8Name, file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	// Normalize timezone to allow us to check for equality
	assert.Equal(fileTime.UTC().Format("010203040506"), file.Timestamp.UTC().Format("010203040506"))
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

func (s *CCLFTestSuite) TestImportCCLF8DBErrors() {
	assert := assert.New(s.T())

	//indeterminate test results without deletion of both.
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A9999")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A9999")

	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0002")
	//postgrestest.DeleteCCLFFilesByCMSID())

	acoID := "A0001"
	fileTime, _ := time.Parse(" ", " ")

	//metadata from success case
	metadata := &cclfFileMetadata{
		name:      constants.CCLF8Name,
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, constants.CCLF8CompPath),
	}
	//Send an invalid context to fail DB check
	ctx, function := context.WithCancel(context.TODO())
	function()

	err := s.importer.importCCLF8(ctx, metadata)
	assert.EqualError(err, "failed to check existence of CCLF8 file: context canceled")

}

func (s *CCLFTestSuite) TestImportCCLF8_alreadyExists() {
	assert := assert.New(s.T())

	hook := test.NewLocal(testUtils.GetLogger(log.API))

	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")

	acoID := "A0001"
	cclfFile := &models.CCLFFile{CCLFNum: 8, ACOCMSID: acoID, Timestamp: time.Now(), PerformanceYear: 18, Name: constants.CCLF8Name}
	postgrestest.CreateCCLFFile(s.T(), s.db, cclfFile)

	metadata := &cclfFileMetadata{
		name:      cclfFile.Name,
		env:       "test",
		acoID:     acoID,
		cclfNum:   cclfFile.CCLFNum,
		perfYear:  cclfFile.PerformanceYear,
		timestamp: cclfFile.Timestamp,
		filePath:  filepath.Join(s.basePath, constants.CCLF8CompPath),
	}

	err := s.importer.importCCLF8(context.Background(), metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	var exists bool
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "already exists in database, skipping import...") {
			exists = true
		}
	}

	assert.True(exists, "CCLF8 file should already exist and should not be imported again.")
}

func (s *CCLFTestSuite) TestImportCCLF8_Invalid() {
	assert := assert.New(s.T())

	var metadata *cclfFileMetadata

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
	err := s.importer.importCCLF8(context.Background(), metadata)
	// This error indicates that we did not supply enough characters for the MBI
	assert.Contains(err.Error(), "invalid byte sequence for encoding \"UTF8\": 0x00")
}

func (s *CCLFTestSuite) TestImportRunoutCCLF() {
	db := database.Connection

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

			s.NoError(s.importer.importCCLF8(context.Background(), metadata))

			cclfFile := postgrestest.GetCCLFFilesByName(s.T(), db, cclfName)[0]
			assert.Equal(t, tt.fileType, cclfFile.Type)
		})
	}
}

func createTemporaryCCLF8ZipFile(t *testing.T, data string) (fileName, cclfName string) {
	cclfName = uuid.New()

	f, err := os.CreateTemp("", "*")
	assert.NoError(t, err)

	w := zip.NewWriter(f)
	f1, err := w.Create(cclfName)
	assert.NoError(t, err)

	_, err = f1.Write([]byte(data))
	assert.NoError(t, err)

	assert.NoError(t, w.Close())

	return f.Name(), cclfName
}
