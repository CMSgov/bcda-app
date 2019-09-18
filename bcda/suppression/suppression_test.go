package suppression

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/models"
)

const BASE_FILE_PATH = "../../shared_files/"

type SuppressionTestSuite struct {
	suite.Suite
}

func (s *SuppressionTestSuite) SetupTest() {
	models.InitializeGormModels()
}

func TestSuppressionTestSuite(t *testing.T) {
	suite.Run(t, new(SuppressionTestSuite))
}

func (s *SuppressionTestSuite) TestImportSuppression() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer db.Close()

	// 181120 file
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &suppressionFileMetadata{
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009",
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009",
		deliveryDate: time.Now(),
	}
	err := importSuppressionData(metadata)
	assert.Nil(err)

	suppressionFile := models.SuppressionFile{}
	db.First(&suppressionFile, "name = ?", metadata.name)
	assert.NotNil(suppressionFile)
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", suppressionFile.Name)
	assert.Equal(fileTime.Format("010203040506"), suppressionFile.Timestamp.Format("010203040506"))
	assert.Equal(constants.ImportComplete, suppressionFile.ImportStatus)

	suppressions := []models.Suppression{}
	db.Find(&suppressions, "file_id = ?", suppressionFile.ID).Order("created_at")
	assert.Len(suppressions, 4)
	assert.Equal("1000087481", suppressions[0].HICN)
	assert.Equal("1-800", suppressions[0].SourceCode)
	assert.Equal("1000093939", suppressions[1].HICN)
	assert.Equal("1-800", suppressions[1].SourceCode)
	assert.Equal("1000079734", suppressions[2].HICN)
	assert.Equal("", suppressions[2].SourceCode)
	assert.Equal("1000050218", suppressions[3].HICN)
	assert.Equal("1-800", suppressions[3].SourceCode)

	err = deleteFilesByFileID(suppressionFile.ID, db)
	assert.Nil(err)

	// 190816 file T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390
	fileTime, _ = time.Parse(time.RFC3339, "2019-08-16T02:41:39Z")
	metadata = &suppressionFileMetadata{
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390",
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390",
		deliveryDate: time.Now(),
	}
	err = importSuppressionData(metadata)
	assert.Nil(err)

	suppressionFile = models.SuppressionFile{}
	db.First(&suppressionFile, "name = ?", metadata.name)
	assert.NotNil(suppressionFile)
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390", suppressionFile.Name)
	assert.Equal(fileTime.Format("010203040506"), suppressionFile.Timestamp.Format("010203040506"))

	suppressions = []models.Suppression{}
	db.Find(&suppressions, "file_id = ?", suppressionFile.ID).Order("created_at")
	assert.Len(suppressions, 250)
	assert.Equal("1000000019", suppressions[0].HICN)
	assert.Equal("N", suppressions[0].PrefIndicator)
	assert.Equal("1000039915", suppressions[20].HICN)
	assert.Equal("N", suppressions[20].PrefIndicator)
	assert.Equal("1000099805", suppressions[100].HICN)
	assert.Equal("N", suppressions[100].PrefIndicator)
	assert.Equal("1000026399", suppressions[200].HICN)
	assert.Equal("N", suppressions[200].PrefIndicator)
	assert.Equal("1000098787", suppressions[249].HICN)
	assert.Equal(" ", suppressions[249].PrefIndicator)

	err = deleteFilesByFileID(suppressionFile.ID, db)
	assert.Nil(err)
}

func (s *SuppressionTestSuite) TestImportSuppression_MissingData() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer db.Close()

	metadata := &suppressionFileMetadata{}
	err := importSuppressionData(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file")

	suppressionFile := models.SuppressionFile{}
	db.First(&suppressionFile, "name = ?", metadata.name)
	assert.NotNil(suppressionFile)
	assert.Equal(constants.ImportFail, suppressionFile.ImportStatus)
	err = deleteFilesByFileID(suppressionFile.ID, db)
	assert.Nil(err)

	filepath := BASE_FILE_PATH + "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000011"
	metadata = &suppressionFileMetadata{
		timestamp:    time.Now(),
		filePath:     filepath,
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000011",
		deliveryDate: time.Now(),
	}
	err = importSuppressionData(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "failed to parse the effective date '20191301' from file: "+filepath)

	suppressionFile = models.SuppressionFile{}
	db.First(&suppressionFile, "name = ?", metadata.name)
	assert.NotNil(suppressionFile)
	assert.Equal(constants.ImportFail, suppressionFile.ImportStatus)
	err = deleteFilesByFileID(suppressionFile.ID, db)
	assert.Nil(err)

	filepath = BASE_FILE_PATH + "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000012"
	metadata = &suppressionFileMetadata{
		timestamp:    time.Now(),
		filePath:     filepath,
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000012",
		deliveryDate: time.Now(),
	}
	err = importSuppressionData(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "failed to parse the samhsa effective date '20191301' from file: "+filepath)

	suppressionFile = models.SuppressionFile{}
	db.First(&suppressionFile, "name = ?", metadata.name)
	assert.NotNil(suppressionFile)
	assert.Equal(constants.ImportFail, suppressionFile.ImportStatus)
	err = deleteFilesByFileID(suppressionFile.ID, db)
	assert.Nil(err)

	filepath = BASE_FILE_PATH + "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000013"
	metadata = &suppressionFileMetadata{
		timestamp:    time.Now(),
		filePath:     filepath,
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000013",
		deliveryDate: time.Now(),
	}
	err = importSuppressionData(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "failed to parse beneficiary link key from file: "+filepath)

	suppressionFile = models.SuppressionFile{}
	db.First(&suppressionFile, "name = ?", metadata.name)
	assert.NotNil(suppressionFile)
	assert.Equal(constants.ImportFail, suppressionFile.ImportStatus)
	err = deleteFilesByFileID(suppressionFile.ID, db)
	assert.Nil(err)
}

func (s *SuppressionTestSuite) TestValidate() {
	assert := assert.New(s.T())

	// positive
	suppressionfilePath := BASE_FILE_PATH + "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"
	metadata := &suppressionFileMetadata{timestamp: time.Now(), filePath: suppressionfilePath}
	err := validate(metadata)
	assert.Nil(err)

	// bad file path
	metadata.filePath = metadata.filePath + "/blah/"
	err = validate(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file "+metadata.filePath)

	// invalid file header
	metadata.filePath = BASE_FILE_PATH + "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"
	err = validate(metadata)
	assert.EqualError(err, "invalid file header for file: "+metadata.filePath)

	// missing record count
	metadata.filePath = BASE_FILE_PATH + "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"
	err = validate(metadata)
	assert.EqualError(err, "failed to parse record count from file: "+metadata.filePath)

	// incorrect record count
	metadata.filePath = BASE_FILE_PATH + "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010"
	err = validate(metadata)
	assert.EqualError(err, "incorrect number of records found from file: '"+metadata.filePath+"'. Expected record count: 5, Actual record count: 4")
}

func (s *SuppressionTestSuite) TestParseMetadata() {
	assert := assert.New(s.T())

	// positive
	expTime, _ := time.Parse(time.RFC3339, "2018-11-20T20:13:01Z")
	metadata, err := parseMetadata("blah/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T2013010")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T2013010", metadata.name)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Nil(err)

	// change the name and timestamp
	expTime, _ = time.Parse(time.RFC3339, "2019-12-20T21:09:42Z")
	metadata, err = parseMetadata("blah/T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420", metadata.name)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Nil(err)
}

func (s *SuppressionTestSuite) TestParseMetadata_InvalidFilename() {
	assert := assert.New(s.T())

	// invalid file name
	_, err := parseMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	_, err = parseMetadata("/path/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000010")
	assert.EqualError(err, "invalid filename for file: /path/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000010")

	// invalid date
	_, err = parseMetadata("/path/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: /path/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420: parsing time \"D190117.T990942\": hour out of range")
}

func (s *SuppressionTestSuite) TestGetSuppressionFileMetadata() {
	assert := assert.New(s.T())
	var suppresslist []*suppressionFileMetadata
	var skipped int
	testUtils.SetPendingDeletionDir(s.Suite)

	filePath := BASE_FILE_PATH + "synthetic1800MedicareFiles/test/"
	testUtils.ResetFiles(s.Suite, filePath)
	err := filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(suppresslist))
	assert.Equal(0, skipped)

	suppresslist = []*suppressionFileMetadata{}
	skipped = 0
	filePath = BASE_FILE_PATH + "suppressionfile_BadFileNames/"
	testUtils.ResetFiles(s.Suite, filePath)
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	suppresslist = []*suppressionFileMetadata{}
	skipped = 0
	filePath = BASE_FILE_PATH + "synthetic1800MedicareFiles/test/"
	testUtils.ResetFiles(s.Suite, filePath)
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	modtimeAfter := time.Now().Truncate(time.Second)
	// check current value and change mod time
	for _, f := range suppresslist {
		fInfo, _ := os.Stat(f.filePath)
		assert.Equal(fInfo.ModTime().Format("010203040506"), f.deliveryDate.Format("010203040506"))

		err = os.Chtimes(f.filePath, modtimeAfter, modtimeAfter)
		if err != nil {
			s.FailNow("Failed to change modified time for file", err)
		}
	}

	suppresslist = []*suppressionFileMetadata{}
	filePath = BASE_FILE_PATH + "synthetic1800MedicareFiles/test/"
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	for _, f := range suppresslist {
		assert.Equal(modtimeAfter.Format("010203040506"), f.deliveryDate.Format("010203040506"))
	}

	testUtils.ResetFiles(s.Suite, filePath)
}

func (s *SuppressionTestSuite) TestGetSuppressionFileMetadata_TimeChange() {
	assert := assert.New(s.T())
	var suppresslist []*suppressionFileMetadata
	var skipped int
	testUtils.SetPendingDeletionDir(s.Suite)
	folderPath := BASE_FILE_PATH + "suppressionfile_BadFileNames/"
	filePath := folderPath + "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	skipped = 0
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)
	testUtils.ResetFiles(s.Suite, folderPath)

	timeChange := origTime.Add(-(time.Hour * 25)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	suppresslist = []*suppressionFileMetadata{}
	skipped = 0
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, "open ../../shared_files/suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009: no such file or directory")

	testUtils.ResetFiles(s.Suite, folderPath)
}

func (s *SuppressionTestSuite) TestCleanupSuppression() {
	assert := assert.New(s.T())
	var suppresslist []*suppressionFileMetadata
	testUtils.SetPendingDeletionDir(s.Suite)

	// failed import: file that's within the threshold - stay put
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:09Z")
	metadata := &suppressionFileMetadata{
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009",
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009",
		imported:     false,
		deliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata2 := &suppressionFileMetadata{
		name:         "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009",
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009",
		imported:     false,
		deliveryDate: fileTime,
	}

	// successful import: should move
	metadata3 := &suppressionFileMetadata{
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		imported:     true,
		deliveryDate: time.Now(),
	}

	suppresslist = []*suppressionFileMetadata{metadata, metadata2, metadata3}
	err := cleanupSuppression(suppresslist)
	assert.Nil(err)

	files, err := ioutil.ReadDir(os.Getenv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", os.Getenv("PENDING_DELETION_DIR"), err)
	}

	for _, file := range files {
		assert.NotEqual("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", file.Name())

		if file.Name() != "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420" && file.Name() != "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009" {
			err = fmt.Errorf("unknown file moved %s", file.Name())
			s.FailNow("test files did not correctly cleanup", err)
		}
	}

	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"suppressionfile_BadFileNames/")
}

func deleteFilesByFileID(fileID uint, db *gorm.DB) error {
	var files []models.SuppressionFile
	db.Where("id = ?", fileID).Find(&files)
	for _, suppressFile := range files {
		err := suppressFile.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}
