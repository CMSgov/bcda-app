package beneprefs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type BenePrefsTestSuite struct {
	suite.Suite
	pendingDeletionDir string

	basePath string
	cleanup  func()
}

func (s *BenePrefsTestSuite) SetupSuite() {
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(&s.Suite, dir)

}

func (s *BenePrefsTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
}

func (s *BenePrefsTestSuite) createImporter(repo models.Repository) BenePrefsImporter {
	logger := log.StandardLogger()
	client := &bcdaaws.MockS3Client{}
	return BenePrefsImporter{
		FileHandler: bcdaaws.S3Helper{
			Client: client,
			Logger: logger,
		},
		Repo:                 repo,
		Logger:               logger,
		ImportStatusInterval: utils.GetEnvInt("SUPPRESS_IMPORT_STATUS_RECORDS_INTERVAL", 1000),
	}
}

func (s *BenePrefsTestSuite) TearDownSuite() {
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *BenePrefsTestSuite) TearDownTest() {
	s.cleanup()
}
func TestBenePrefsTestSuite(t *testing.T) {
	suite.Run(t, new(BenePrefsTestSuite))
}

func (s *BenePrefsTestSuite) TestImportFile() {
	assert := assert.New(s.T())
	ctx := context.Background()

	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &models.BenePrefsFilenameMetadata{
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"),
		Name:         constants.TestSuppressMetaFileName,
		DeliveryDate: time.Now(),
	}

	// happy path
	repo := &models.MockRepository{}
	repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(uint(1), nil)
	repo.On("UpdateBenePrefsImportStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBenePrefsRecord", mock.Anything, mock.Anything).Return(nil)
	importer := s.createImporter(repo)
	err := importer.importFile(ctx, metadata)
	assert.Nil(err)

	// issue saving the bene-prefs file record
	repo = &models.MockRepository{}
	repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(uint(1), errors.New("throw db error"))
	importer = s.createImporter(repo)
	err = importer.importFile(ctx, metadata)
	assert.ErrorContains(err, "failed to create bene-prefs file record for file")
	assert.ErrorContains(err, "throw db error")

	// issue updating the bene-prefs file record status
	repo = &models.MockRepository{}
	repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(uint(1), nil)
	repo.On("UpdateBenePrefsImportStatus", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("throw db error"))
	repo.On("CreateBenePrefsRecord", mock.Anything, mock.Anything).Return(nil)
	importer = s.createImporter(repo)
	err = importer.importFile(ctx, metadata)
	assert.ErrorContains(err, "could not update bene-prefs file import status for file")
	assert.ErrorContains(err, "throw db error")

	// issue saving the bene-prefs record
	repo = &models.MockRepository{}
	repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(uint(1), nil)
	repo.On("UpdateBenePrefsImportStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBenePrefsRecord", mock.Anything, mock.Anything).Return(errors.New("throw db error"))
	importer = s.createImporter(repo)
	err = importer.importFile(ctx, metadata)
	assert.ErrorContains(err, "failed to create bene-prefs record")
	assert.ErrorContains(err, "throw db error")
}

func (s *BenePrefsTestSuite) TestImport_MissingData() {
	assert := assert.New(s.T())
	ctx := context.Background()
	testUint := uint(0)

	// Verify empty file is rejected
	metadata := &models.BenePrefsFilenameMetadata{}
	repo := &models.MockRepository{}
	repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(testUint, nil)
	repo.On("UpdateBenePrefsImportStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	importer := s.createImporter(repo)
	err := importer.importFile(ctx, metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file")

	tests := []struct {
		name    string
		expErr  string
		dbError bool
	}{
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000011", "failed to parse the effective date '20191301' from file", false},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000012", "failed to parse the samhsa effective date '20191301' from file", false},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000013", "failed to parse beneficiary link key from file", false},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000011", "failed to create bene-prefs file record for file", true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fp := filepath.Join(s.basePath, "suppressionfile_MissingData/"+tt.name)
			metadata = &models.BenePrefsFilenameMetadata{
				Timestamp:    time.Now(),
				FilePath:     fp,
				Name:         tt.name,
				DeliveryDate: time.Now(),
			}

			repo := &models.MockRepository{}
			repo.On("UpdateBenePrefsImportStatus", mock.Anything, mock.Anything, constants.ImportFail).Return(nil)
			if tt.dbError {
				repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(testUint, errors.New("throw db error"))
			} else {
				repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(testUint, nil)
			}
			importer := s.createImporter(repo)

			err = importer.importFile(ctx, metadata)
			assert.NotNil(err)
			assert.ErrorContains(err, fmt.Sprintf("%s: %s", tt.expErr, fp))
		})
	}
}

func (s *BenePrefsTestSuite) TestValidate() {
	assert := assert.New(s.T())
	repo := &models.MockRepository{}
	importer := s.createImporter(repo)
	ctx := context.Background()

	// positive
	suppressionfilePath := filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	metadata := &models.BenePrefsFilenameMetadata{Timestamp: time.Now(), FilePath: suppressionfilePath}
	err := importer.validate(ctx, metadata)
	assert.Nil(err)

	// bad file path
	metadata.FilePath = metadata.FilePath + "/blah/"
	err = importer.validate(ctx, metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file "+metadata.FilePath)

	// invalid file header
	metadata.FilePath = filepath.Join(s.basePath, "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	err = importer.validate(ctx, metadata)
	assert.EqualError(err, "invalid file header for file: "+metadata.FilePath)

	// missing record count
	metadata.FilePath = filepath.Join(s.basePath, "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	err = importer.validate(ctx, metadata)
	assert.EqualError(err, "failed to parse record count from file: "+metadata.FilePath)

	// incorrect record count
	metadata.FilePath = filepath.Join(s.basePath, "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010")
	err = importer.validate(ctx, metadata)
	assert.EqualError(err, "incorrect number of records found from file: '"+metadata.FilePath+"'. Expected record count: 5, Actual record count: 4")
}

func (s *BenePrefsTestSuite) TestLoadBenePrefsFiles() {
	assert := assert.New(s.T())
	repo := &models.MockRepository{}
	importer := s.createImporter(repo)
	ctx := context.Background()

	filePath := filepath.Join(s.basePath, constants.TestSynthMedFilesPath)
	suppresslist, skipped, err := LoadBenePrefsFiles(ctx, importer.FileHandler, filePath)
	assert.Nil(err)
	assert.Equal(2, len(*suppresslist))
	assert.Equal(0, skipped)

	filePath = filepath.Join(s.basePath, "suppressionfile_BadFileNames/")
	suppresslist, skipped, err = LoadBenePrefsFiles(ctx, importer.FileHandler, filePath)
	assert.Nil(err)
	assert.Equal(0, len(*suppresslist))
	assert.Equal(2, skipped)

	filePath = filepath.Join(s.basePath, constants.TestSynthMedFilesPath)
	suppresslist, _, err = LoadBenePrefsFiles(ctx, importer.FileHandler, filePath)
	assert.Nil(err)
	modtimeAfter := time.Now().Truncate(time.Second)
	// check current value and change mod time
	for _, f := range *suppresslist {
		fInfo, _ := os.Stat(f.FilePath)
		assert.Equal(fInfo.ModTime().Format("010203040506"), f.DeliveryDate.Format("010203040506"))

		err = os.Chtimes(f.FilePath, modtimeAfter, modtimeAfter)
		if err != nil {
			s.FailNow(constants.TestChangeTimeErr, err)
		}
	}

	filePath = filepath.Join(s.basePath, constants.TestSynthMedFilesPath)
	suppresslist, _, err = LoadBenePrefsFiles(ctx, importer.FileHandler, filePath)
	assert.Nil(err)
	for _, f := range *suppresslist {
		assert.Equal(modtimeAfter.Format("010203040506"), f.DeliveryDate.Format("010203040506"))
	}
}

func (s *BenePrefsTestSuite) TestLoadBenePrefsFiles_TimeChange() {
	assert := assert.New(s.T())
	ctx := context.Background()
	repo := &models.MockRepository{}
	importer := s.createImporter(repo)
	// importer.Repo = *postgres.NewRepository(database.Connect())

	folderPath := filepath.Join(s.basePath, "suppressionfile_BadFileNames/")
	filePath := filepath.Join(folderPath, constants.TestSuppressBadPath)

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	suppresslist, skipped, err := LoadBenePrefsFiles(ctx, importer.FileHandler, folderPath)
	assert.Nil(err)
	assert.Equal(0, len(*suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)

	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	suppresslist, skipped, err = LoadBenePrefsFiles(ctx, importer.FileHandler, folderPath)
	assert.Nil(err)
	assert.Equal(0, len(*suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, fmt.Sprintf("open %s: no such file or directory", filePath))

	//Utilize the other bad file, but set an invalid pending deletion directory.
	filePath = filepath.Join(folderPath, constants.TestSuppressBadDeletePath)
	_, err = os.Open(filePath)
	assert.Nil(err)

	timeChange = origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)

	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	// importer.FileHandler.(*LocalFileHandler).PendingDeletionDir = "\n"
	_, _, err = LoadBenePrefsFiles(ctx, importer.FileHandler, folderPath)
	assert.Equal(true, strings.Contains(err.Error(), "error moving unknown file"))
}

func (s *BenePrefsTestSuite) TestCleanupSuppression() {
	assert := assert.New(s.T())
	ctx := context.Background()
	repo := &models.MockRepository{}
	importer := s.createImporter(repo)

	var suppresslist []*models.BenePrefsFilenameMetadata

	// failed import: file that's within the threshold - stay put
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:09Z")
	metadata := &models.BenePrefsFilenameMetadata{
		Name:         constants.TestSuppressMetaFileName,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"),
		Imported:     false,
		DeliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata2 := &models.BenePrefsFilenameMetadata{
		Name:         constants.TestSuppressBadPath,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		Imported:     false,
		DeliveryDate: fileTime,
	}

	// successful import: should move
	metadata3 := &models.BenePrefsFilenameMetadata{
		Name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420"),
		Imported:     true,
		DeliveryDate: time.Now(),
	}

	suppresslist = []*models.BenePrefsFilenameMetadata{metadata, metadata2, metadata3}
	err := CleanupBenePrefsFiles(ctx, importer.FileHandler, suppresslist)
	assert.Nil(err)

	files, err := os.ReadDir(conf.GetEnv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
	}

	for _, file := range files {
		assert.NotEqual(constants.TestSuppressMetaFileName, file.Name())

		if file.Name() != "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420" && file.Name() != constants.TestSuppressBadPath {
			err = fmt.Errorf("unknown file moved %s", file.Name())
			s.FailNow("test files did not correctly cleanup", err)
		}
	}
}

func (s *BenePrefsTestSuite) TestCleanupSuppression_Bad() {
	assert := assert.New(s.T())
	ctx := context.Background()
	repo := &models.MockRepository{}
	importer := s.createImporter(repo)
	// importer.FileHandler.(*LocalFileHandler).PendingDeletionDir = "\n"

	var suppresslist []*models.BenePrefsFilenameMetadata

	//new use cases
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata1 := &models.BenePrefsFilenameMetadata{
		Name:         constants.TestSuppressBadPath,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		Imported:     false,
		DeliveryDate: fileTime,
	}

	//
	metadata2 := &models.BenePrefsFilenameMetadata{
		Name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420"),
		Imported:     true,
		DeliveryDate: time.Now(),
	}

	suppresslist = []*models.BenePrefsFilenameMetadata{metadata1, metadata2}
	err := CleanupBenePrefsFiles(ctx, importer.FileHandler, suppresslist)
	assert.EqualError(err, "2 files could not be cleaned up")
}

func (s *BenePrefsTestSuite) TestCleanupSuppression_RenameFileError() {
	assert := assert.New(s.T())
	ctx := context.Background()
	repo := &models.MockRepository{}
	importer := s.createImporter(repo)
	// importer.FileHandler.(*LocalFileHandler).PendingDeletionDir = "\n"

	var suppresslist []*models.BenePrefsFilenameMetadata

	//Induce an error when attempting to rename file
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata1 := &models.BenePrefsFilenameMetadata{
		Name:         constants.TestSuppressBadPath,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		Imported:     false,
		DeliveryDate: fileTime,
	}

	suppresslist = []*models.BenePrefsFilenameMetadata{metadata1}
	err := CleanupBenePrefsFiles(ctx, importer.FileHandler, suppresslist)
	assert.EqualError(err, "1 files could not be cleaned up")
}

func (s *BenePrefsTestSuite) TestImportDirectoryTable() {
	assert := assert.New(s.T())
	ctx := context.Background()

	tests := []struct {
		name           string
		directory      string
		success        int
		failure        int
		skipped        int
		errorExpected  bool
		errMessage     string
		insertCarriage bool
	}{
		{name: "Valid test", directory: "../../shared_files/synthetic1800MedicareFiles/test2/", success: 2, failure: 0, skipped: 0, errorExpected: false, errMessage: ""},
		{name: "Import failure", directory: "../../shared_files/suppressionfile_BadHeader/", success: 0, failure: 1, skipped: 0, errorExpected: true, errMessage: "one or more bene-prefs files failed to import correctly"},
		{name: "Skipped import", directory: "../../shared_files/suppressionfile_BadFileNames/", success: 0, failure: 0, skipped: 2, errorExpected: false, errMessage: ""},
		{name: "Carriage char in path", directory: "../../shared_files/suppressionfile_BadFileNames/", success: 0, failure: 0, skipped: 0, errorExpected: true, errMessage: "no such file or directory", insertCarriage: true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), tt.directory)
			defer cleanup()
			if tt.insertCarriage {
				path += "\n"
			}

			repo := &models.MockRepository{}
			if !tt.errorExpected {
				repo.On("CreateBenePrefsFile", mock.Anything, mock.Anything).Return(uint(1), nil)
				repo.On("UpdateBenePrefsImportStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				repo.On("CreateBenePrefsRecord", mock.Anything, mock.Anything).Return(nil)
			}

			importer := s.createImporter(repo)
			success, failure, skipped, err := importer.ImportDirectory(ctx, path)
			if tt.errorExpected {
				assert.ErrorContains(err, tt.errMessage)
			} else {
				assert.Nil(err)
			}
			assert.Equal(tt.success, success)
			assert.Equal(tt.failure, failure)
			assert.Equal(tt.skipped, skipped)
		})
	}
}
