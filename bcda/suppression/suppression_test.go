package suppression

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SuppressionTestSuite struct {
	suite.Suite
	pendingDeletionDir string

	basePath string
	cleanup  func()
}

func (s *SuppressionTestSuite) SetupSuite() {
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)

}

func (s *SuppressionTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
}

func (s *SuppressionTestSuite) TearDownSuite() {
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *SuppressionTestSuite) TearDownTest() {
	s.cleanup()
}
func TestSuppressionTestSuite(t *testing.T) {
	suite.Run(t, new(SuppressionTestSuite))
}

func (s *SuppressionTestSuite) TestImportSuppression() {
	assert := assert.New(s.T())
	db := database.Connection

	// 181120 file
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &optout.OptOutFilenameMetadata{
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"),
		Name:         constants.TestSuppressMetaFileName,
		DeliveryDate: time.Now(),
	}
	err := importSuppressionData(metadata)
	assert.Nil(err)

	suppressionFile := postgrestest.GetSuppressionFileByName(s.T(), db, metadata.Name)[0]
	assert.Equal(constants.TestSuppressMetaFileName, suppressionFile.Name)
	assert.Equal(fileTime.Format("010203040506"), suppressionFile.Timestamp.UTC().Format("010203040506"))
	assert.Equal(constants.ImportComplete, suppressionFile.ImportStatus)

	suppressions := postgrestest.GetSuppressionsByFileID(s.T(), db, suppressionFile.ID)
	assert.Len(suppressions, 4)
	assert.Equal("5SJ0A00AA00", suppressions[0].MBI)
	assert.Equal("1-800", suppressions[0].SourceCode)
	assert.Equal("4SF6G00AA00", suppressions[1].MBI)
	assert.Equal("1-800", suppressions[1].SourceCode)
	assert.Equal("4SH0A00AA00", suppressions[2].MBI)
	assert.Equal("", suppressions[2].SourceCode)
	assert.Equal("8SG0A00AA00", suppressions[3].MBI)
	assert.Equal("1-800", suppressions[3].SourceCode)

	postgrestest.DeleteSuppressionFileByID(s.T(), db, suppressionFile.ID)

	// 190816 file T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390
	fileTime, _ = time.Parse(time.RFC3339, "2019-08-16T02:41:39Z")
	metadata = &optout.OptOutFilenameMetadata{
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390"),
		Name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390",
		DeliveryDate: time.Now(),
	}
	err = importSuppressionData(metadata)
	assert.Nil(err)

	suppressionFile = postgrestest.GetSuppressionFileByName(s.T(), db, metadata.Name)[0]
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390", suppressionFile.Name)
	assert.Equal(fileTime.Format("010203040506"), suppressionFile.Timestamp.UTC().Format("010203040506"))

	suppressions = postgrestest.GetSuppressionsByFileID(s.T(), db, suppressionFile.ID)
	assert.Len(suppressions, 250)
	assert.Equal("1000000019", suppressions[0].MBI)
	assert.Equal("N", suppressions[0].PrefIndicator)
	assert.Equal("1000039915", suppressions[20].MBI)
	assert.Equal("N", suppressions[20].PrefIndicator)
	assert.Equal("1000099805", suppressions[100].MBI)
	assert.Equal("N", suppressions[100].PrefIndicator)
	assert.Equal("1000026399", suppressions[200].MBI)
	assert.Equal("N", suppressions[200].PrefIndicator)
	assert.Equal("1000098787", suppressions[249].MBI)
	assert.Equal(" ", suppressions[249].PrefIndicator)

	postgrestest.DeleteSuppressionFileByID(s.T(), db, suppressionFile.ID)
}

func (s *SuppressionTestSuite) TestImportSuppression_MissingData() {
	assert := assert.New(s.T())
	db := database.Connection

	// Verify empty file is rejected
	metadata := &optout.OptOutFilenameMetadata{}
	err := importSuppressionData(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file")

	tests := []struct {
		name   string
		expErr string
	}{
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000011", "failed to parse the effective date '20191301' from file"},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000012", "failed to parse the samhsa effective date '20191301' from file"},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000013", "failed to parse beneficiary link key from file"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fp := filepath.Join(s.basePath, "suppressionfile_MissingData/"+tt.name)
			metadata = &optout.OptOutFilenameMetadata{
				Timestamp:    time.Now(),
				FilePath:     fp,
				Name:         tt.name,
				DeliveryDate: time.Now(),
			}
			err = importSuppressionData(metadata)
			assert.NotNil(err)
			assert.Contains(err.Error(), fmt.Sprintf("%s: %s", tt.expErr, fp))

			suppressionFile := postgrestest.GetSuppressionFileByName(s.T(), db, metadata.Name)[0]
			assert.Equal(constants.ImportFail, suppressionFile.ImportStatus)
			postgrestest.DeleteSuppressionFileByID(s.T(), db, suppressionFile.ID)
		})
	}
}

func (s *SuppressionTestSuite) TestValidate() {
	assert := assert.New(s.T())

	// positive
	suppressionfilePath := filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	metadata := &optout.OptOutFilenameMetadata{Timestamp: time.Now(), FilePath: suppressionfilePath}
	err := validate(metadata)
	assert.Nil(err)

	// bad file path
	metadata.FilePath = metadata.FilePath + "/blah/"
	err = validate(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file "+metadata.FilePath)

	// invalid file header
	metadata.FilePath = filepath.Join(s.basePath, "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	err = validate(metadata)
	assert.EqualError(err, "invalid file header for file: "+metadata.FilePath)

	// missing record count
	metadata.FilePath = filepath.Join(s.basePath, "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	err = validate(metadata)
	assert.EqualError(err, "failed to parse record count from file: "+metadata.FilePath)

	// incorrect record count
	metadata.FilePath = filepath.Join(s.basePath, "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010")
	err = validate(metadata)
	assert.EqualError(err, "incorrect number of records found from file: '"+metadata.FilePath+"'. Expected record count: 5, Actual record count: 4")
}
func (s *SuppressionTestSuite) TestGetSuppressionFileMetadata() {
	assert := assert.New(s.T())
	var suppresslist []*optout.OptOutFilenameMetadata
	var skipped int

	filePath := filepath.Join(s.basePath, constants.TestSynthMedFilesPath)
	err := filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(suppresslist))
	assert.Equal(0, skipped)

	suppresslist = []*optout.OptOutFilenameMetadata{}
	skipped = 0
	filePath = filepath.Join(s.basePath, "suppressionfile_BadFileNames/")
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	suppresslist = []*optout.OptOutFilenameMetadata{}
	skipped = 0
	filePath = filepath.Join(s.basePath, constants.TestSynthMedFilesPath)
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	modtimeAfter := time.Now().Truncate(time.Second)
	// check current value and change mod time
	for _, f := range suppresslist {
		fInfo, _ := os.Stat(f.FilePath)
		assert.Equal(fInfo.ModTime().Format("010203040506"), f.DeliveryDate.Format("010203040506"))

		err = os.Chtimes(f.FilePath, modtimeAfter, modtimeAfter)
		if err != nil {
			s.FailNow(constants.TestChangeTimeErr, err)
		}
	}

	suppresslist = []*optout.OptOutFilenameMetadata{}
	filePath = filepath.Join(s.basePath, constants.TestSynthMedFilesPath)
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	for _, f := range suppresslist {
		assert.Equal(modtimeAfter.Format("010203040506"), f.DeliveryDate.Format("010203040506"))
	}
}

func (s *SuppressionTestSuite) TestGetSuppressionFileMetadata_TimeChange() {
	assert := assert.New(s.T())
	testUtils.SetPendingDeletionDir(s.Suite, s.pendingDeletionDir)
	var suppresslist []*optout.OptOutFilenameMetadata
	var skipped int
	folderPath := filepath.Join(s.basePath, "suppressionfile_BadFileNames/")
	filePath := filepath.Join(folderPath, constants.TestSuppressBadPath)

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	skipped = 0
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)

	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	suppresslist = []*optout.OptOutFilenameMetadata{}
	skipped = 0
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
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

	suppresslist = []*optout.OptOutFilenameMetadata{}
	conf.SetEnv(s.T(), "PENDING_DELETION_DIR", "\n")
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Equal(true, strings.Contains(err.Error(), "error moving unknown file"))

}

func (s *SuppressionTestSuite) TestCleanupSuppression() {
	assert := assert.New(s.T())
	var suppresslist []*optout.OptOutFilenameMetadata

	// failed import: file that's within the threshold - stay put
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:09Z")
	metadata := &optout.OptOutFilenameMetadata{
		Name:         constants.TestSuppressMetaFileName,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"),
		Imported:     false,
		DeliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata2 := &optout.OptOutFilenameMetadata{
		Name:         constants.TestSuppressBadPath,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		Imported:     false,
		DeliveryDate: fileTime,
	}

	// successful import: should move
	metadata3 := &optout.OptOutFilenameMetadata{
		Name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420"),
		Imported:     true,
		DeliveryDate: time.Now(),
	}

	suppresslist = []*optout.OptOutFilenameMetadata{metadata, metadata2, metadata3}
	err := cleanupSuppression(suppresslist)
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

func (s *SuppressionTestSuite) TestCleanupSuppression_Bad() {
	assert := assert.New(s.T())
	var suppresslist []*optout.OptOutFilenameMetadata

	//new use cases
	conf.SetEnv(s.T(), "PENDING_DELETION_DIR", "\n")
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata1 := &optout.OptOutFilenameMetadata{
		Name:         constants.TestSuppressBadPath,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		Imported:     false,
		DeliveryDate: fileTime,
	}

	//
	metadata2 := &optout.OptOutFilenameMetadata{
		Name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420"),
		Imported:     true,
		DeliveryDate: time.Now(),
	}

	suppresslist = []*optout.OptOutFilenameMetadata{metadata1, metadata2}
	err := cleanupSuppression(suppresslist)
	assert.EqualError(err, "2 files could not be cleaned up")

}

func (s *SuppressionTestSuite) TestCleanupSuppression_RenameFileError() {
	assert := assert.New(s.T())
	var suppresslist []*optout.OptOutFilenameMetadata

	//Induce an error when attempting to rename file
	conf.SetEnv(s.T(), "PENDING_DELETION_DIR", "\n")
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata1 := &optout.OptOutFilenameMetadata{
		Name:         constants.TestSuppressBadPath,
		Timestamp:    fileTime,
		FilePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		Imported:     false,
		DeliveryDate: fileTime,
	}

	suppresslist = []*optout.OptOutFilenameMetadata{metadata1}
	err := cleanupSuppression(suppresslist)
	assert.EqualError(err, "1 files could not be cleaned up")

}

func (s *SuppressionTestSuite) TestImportSuppressionDirectoryTable() {
	assert := assert.New(s.T())
	db := database.Connection

	tests := []struct {
		name           string
		directory      string
		success        int
		failure        int
		skipped        int
		errorExpected  bool
		errMessage     string
		deleteFiles    bool
		insertCarriage bool
	}{
		{name: "Valid test", directory: "../../shared_files/synthetic1800MedicareFiles/test2/", success: 2, failure: 0, skipped: 0, errorExpected: false, errMessage: "", deleteFiles: true},
		{name: "Import failure", directory: "../../shared_files/suppressionfile_BadHeader/", success: 0, failure: 1, skipped: 0, errorExpected: true, errMessage: "one or more suppression files failed to import correctly", deleteFiles: false},
		{name: "Skipped import", directory: "../../shared_files/suppressionfile_BadFileNames/", success: 0, failure: 0, skipped: 2, errorExpected: false, errMessage: "", deleteFiles: false},
		{name: "Carriage char in path", directory: "../../shared_files/suppressionfile_BadFileNames/", success: 0, failure: 0, skipped: 0, errorExpected: true, errMessage: "no such file or directory", deleteFiles: false, insertCarriage: true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {

			path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), tt.directory)
			defer cleanup()
			if tt.insertCarriage {
				path += "\n"
			}

			success, failure, skipped, err := ImportSuppressionDirectory(path)
			if tt.errorExpected {
				assert.Equal(true, strings.Contains(err.Error(), tt.errMessage))
			} else {
				assert.Nil(err)
			}
			assert.Equal(tt.success, success)
			assert.Equal(tt.failure, failure)
			assert.Equal(tt.skipped, skipped)

			if tt.deleteFiles {
				fs := postgrestest.GetSuppressionFileByName(s.T(), db,
					"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010",
					"T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241391")
				assert.Len(fs, 2)
				for _, f := range fs {
					postgrestest.DeleteSuppressionFileByID(s.T(), db, f.ID)
				}
			}

		})
	}
}
