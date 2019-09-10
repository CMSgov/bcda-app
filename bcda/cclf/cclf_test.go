package cclf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

const BASE_FILE_PATH = "../../shared_files/"

type CCLFTestSuite struct {
	suite.Suite
}

func (s *CCLFTestSuite) SetupTest() {
	models.InitializeGormModels()
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFTestSuite))
}

func (s *CCLFTestSuite) TestImportCCLF0() {
	assert := assert.New(s.T())

	cclf0filePath := BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}

	// positive
	validator, err := importCCLF0(cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])

	// negative
	cclf0metadata = &cclfFileMetadata{}
	_, err = importCCLF0(cclf0metadata)
	assert.EqualError(err, "could not read CCLF0 archive : read .: is a directory")

	// missing cclf8 from cclf0
	cclf0filePath = BASE_FILE_PATH + "cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.EqualError(err, "failed to parse CCLF8 from CCLF0 file ../../shared_files/cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000011")

	// duplicate file types from cclf0
	cclf0filePath = BASE_FILE_PATH + "cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000013"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.EqualError(err, "duplicate CCLF8 file type found from CCLF0 file")
}

func (s *CCLFTestSuite) TestImportCCLF0_SplitFiles() {
	assert := assert.New(s.T())

	cclf0filePath := BASE_FILE_PATH + "cclf_split/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}

	validator, err := importCCLF0(cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])
}

func (s *CCLFTestSuite) TestValidate() {
	assert := assert.New(s.T())

	cclf8filePath := BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"
	cclf8metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18}

	// positive
	cclfvalidator := map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 6, maxRecordLength: 549}}
	err := validate(cclf8metadata, cclfvalidator)
	assert.Nil(err)

	// negative
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 2, maxRecordLength: 549}}
	err = validate(cclf8metadata, cclfvalidator)
	assert.EqualError(err, "maximum record count reached for file CCLF8 (expected: 2, actual: 3)")
}

func (s *CCLFTestSuite) TestValidate_SplitFiles() {
	assert := assert.New(s.T())

	cclf8Metadata := &cclfFileMetadata{
		acoID:     "A0001",
		cclfNum:   8,
		timestamp: time.Now(),
		filePath:  BASE_FILE_PATH + "cclf_split/T.A0001.ACO.ZC8Y18.D181120.T1000009",
		perfYear:  18,
	}

	validator := map[string]cclfFileValidator{
		"CCLF8": {totalRecordCount: 6, maxRecordLength: 549},
	}

	err := validate(cclf8Metadata, validator)
	assert.Nil(err)
}

func (s *CCLFTestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err := deleteFilesByACO("A0001", db)
	assert.Nil(err)

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &cclfFileMetadata{
		name:      "T.A0001.ACO.ZC8Y18.D181120.T1000009",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009",
	}

	err = importCCLF8(metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := models.CCLFFile{}
	db.First(&file, "name = ?", metadata.name)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC8Y18.D181120.T1000009", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime.Format("010203040506"), file.Timestamp.Format("010203040506"))
	assert.Equal(18, file.PerformanceYear)
	assert.Equal(constants.ImportComplete, file.ImportStatus)

	beneficiaries := []models.CCLFBeneficiary{}
	db.Find(&beneficiaries, "file_id = ?", file.ID)
	assert.Equal(6, len(beneficiaries))
	assert.Equal("203031401M", beneficiaries[0].HICN)
	assert.Equal("1A69B98CD30", beneficiaries[0].MBI)
	assert.Equal("203031402A", beneficiaries[1].HICN)
	assert.Equal("1A69B98CD31", beneficiaries[1].MBI)
	assert.Equal("203031403A", beneficiaries[2].HICN)
	assert.Equal("1A69B98CD32", beneficiaries[2].MBI)
	assert.Equal("203031404A", beneficiaries[3].HICN)
	assert.Equal("1A69B98CD33", beneficiaries[3].MBI)
	assert.Equal("203031405C7", beneficiaries[4].HICN)
	assert.Equal("1A69B98CD34", beneficiaries[4].MBI)
	assert.Equal("203031406M", beneficiaries[5].HICN)
	assert.Equal("1A69B98CD35", beneficiaries[5].MBI)

	err = deleteFilesByACO("A0001", db)
	assert.Nil(err)
}

func (s *CCLFTestSuite) TestImportCCLF8_SplitFiles() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err := deleteFilesByACO("A0001", db)
	assert.Nil(err)

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &cclfFileMetadata{
		name:      "T.A0001.ACO.ZC8Y18.D181120.T1000009",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  BASE_FILE_PATH + "cclf_split/T.A0001.ACO.ZC8Y18.D181120.T1000009",
	}

	err = importCCLF8(metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := models.CCLFFile{}
	db.First(&file, "name = ?", metadata.name)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC8Y18.D181120.T1000009", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime.Format("010203040506"), file.Timestamp.Format("010203040506"))
	assert.Equal(18, file.PerformanceYear)
	assert.Equal(constants.ImportComplete, file.ImportStatus)

	beneficiaries := []models.CCLFBeneficiary{}
	db.Find(&beneficiaries, "file_id = ?", file.ID)
	assert.Equal(6, len(beneficiaries))
	assert.Equal("203031401M", beneficiaries[0].HICN)
	assert.Equal("1A69B98CD30", beneficiaries[0].MBI)
	assert.Equal("203031402A", beneficiaries[1].HICN)
	assert.Equal("1A69B98CD31", beneficiaries[1].MBI)
	assert.Equal("203031403A", beneficiaries[2].HICN)
	assert.Equal("1A69B98CD32", beneficiaries[2].MBI)
	assert.Equal("203031404A", beneficiaries[3].HICN)
	assert.Equal("1A69B98CD33", beneficiaries[3].MBI)
	assert.Equal("203031405C7", beneficiaries[4].HICN)
	assert.Equal("1A69B98CD34", beneficiaries[4].MBI)
	assert.Equal("203031406M", beneficiaries[5].HICN)
	assert.Equal("1A69B98CD35", beneficiaries[5].MBI)

	err = deleteFilesByACO("A0001", db)
	assert.Nil(err)
}

func (s *CCLFTestSuite) TestImportCCLF8_InvalidMetadata() {
	assert := assert.New(s.T())

	var metadata *cclfFileMetadata
	err := importCCLF8(metadata)
	assert.EqualError(err, "CCLF file not found")
}

func (s *CCLFTestSuite) TestGetCCLFFileMetadata() {
	assert := assert.New(s.T())

	expTime, _ := time.Parse(time.RFC3339, "2019-01-19T20:13:01Z")
	metadata, err := getCCLFFileMetadata("/path/T.A0002.ACO.ZC0Y18.D190119.T2013010")
	assert.Equal("test", metadata.env)
	assert.Equal("A0002", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	expTime, _ = time.Parse(time.RFC3339, "2019-01-17T21:09:42Z")
	metadata, err = getCCLFFileMetadata("/path/T.T0000.ACO.ZC8Y18.D190117.T2109420")
	assert.Equal("test", metadata.env)
	assert.Equal("T0000", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	expTime, _ = time.Parse(time.RFC3339, "2019-01-08T23:55:00Z")
	metadata, err = getCCLFFileMetadata("/path/P.A0001.ACO.ZC9Y18.D190108.T2355000")
	assert.EqualError(err, "invalid filename for file: /path/P.A0001.ACO.ZC9Y18.D190108.T2355000")

	// CMS EFT file format structure
	expTime, _ = time.Parse(time.RFC3339, "2019-01-19T20:13:01Z")
	metadata, err = getCCLFFileMetadata("/cclf/T#EFT.ON.A0001.ACOB.ZC0Y19.D190119.T2013010")
	assert.Equal("test", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(19, metadata.perfYear)
	assert.Nil(err)

	// CMS EFT file format with BCD identifier
	metadata, err = getCCLFFileMetadata("/BCD/T.BCD.ACO.ZC0Y18.D181120.T0001000")
	date := fmt.Sprintf("D181120.T%s", time.Now().Format("150405"))
	timestamp, _ := time.Parse("D060102.T150405", date)
	assert.Equal("test", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(timestamp.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	metadata, err = getCCLFFileMetadata("/BCD/T.BCD.ACO.ZC8Y18.D190112.T0012000")
	date = fmt.Sprintf("D190112.T%s", time.Now().Format("150405"))
	timestamp, _ = time.Parse("D060102.T150405", date)
	assert.Equal("test", metadata.env)
	assert.Equal("A0012", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(timestamp.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	metadata, err = getCCLFFileMetadata("/BCD/P.BCD.ACO.ZC9Y19.D180610.T0002000")
	assert.EqualError(err, "invalid filename for file: /BCD/P.BCD.ACO.ZC9Y19.D180610.T0002000")
}

func (s *CCLFTestSuite) TestGetCCLFFileMetadata_InvalidFilename() {
	assert := assert.New(s.T())

	_, err := getCCLFFileMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	_, err = getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: /path/T.A0000.ACO.ZC8Y18.D190117.T9909420: parsing time \"D190117.T990942\": hour out of range")

	_, err = getCCLFFileMetadata("/cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")
	assert.EqualError(err, "invalid filename for file: /cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")

	_, err = getCCLFFileMetadata("/cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
	assert.EqualError(err, "invalid filename for file: /cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
}

func (s *CCLFTestSuite) TestSortCCLFFiles() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[int][]*cclfFileMetadata)
	var skipped int

	filePath := BASE_FILE_PATH + "cclf/"
	err := filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(cclfmap["A0001"][18]))
	assert.Equal(1, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf_BCD/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(cclfmap["A0001"][18]))
	assert.Equal(1, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf_BadFileNames/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist := cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist))
	assert.Equal(3, skipped)
	for _, cclf := range cclflist {
		assert.NotEqual(9, cclf.cclfNum)
	}
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf0/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(3, len(cclflist))
	assert.Equal(0, skipped)
	for _, cclf := range cclflist {
		assert.Equal(0, cclf.cclfNum)
	}

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf8/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(5, len(cclflist))
	assert.Equal(0, skipped)
	for _, cclf := range cclflist {
		assert.Equal(8, cclf.cclfNum)
	}

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf9/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(0, len(cclflist))
	assert.Equal(4, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf_All/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(8, len(cclflist))
	assert.Equal(4, skipped)
	var cclf0, cclf8, cclf9 []*cclfFileMetadata

	for _, cclf := range cclflist {
		if cclf.cclfNum == 0 {
			cclf0 = append(cclf0, cclf)
		} else if cclf.cclfNum == 8 {
			cclf8 = append(cclf8, cclf)
		} else if cclf.cclfNum == 9 {
			cclf9 = append(cclf9, cclf)
		}
	}
	assert.Equal(5, len(cclf8))
	assert.Equal(0, len(cclf9))
	assert.Equal(3, len(cclf0))
	testUtils.ResetFiles(s.Suite, filePath)
}

func (s *CCLFTestSuite) TestSortCCLFFiles_TimeChange() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[int][]*cclfFileMetadata)
	var skipped int
	folderPath := BASE_FILE_PATH + "cclf_BadFileNames/"
	filePath := folderPath + "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	skipped = 0
	err = filepath.Walk(folderPath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist := cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist))
	assert.Equal(3, skipped)
	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf_BadFileNames/")

	timeChange := origTime.Add(-(time.Hour * 25)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	err = filepath.Walk(folderPath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist))
	assert.Equal(3, skipped)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, "open ../../shared_files/cclf_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009: no such file or directory")

	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf_BadFileNames/")
}

func (s *CCLFTestSuite) TestSortCCLFFiles_InvalidPath() {
	cclfMap := make(map[string]map[int][]*cclfFileMetadata)
	skipped := 0
	err := filepath.Walk("./foo", sortCCLFFiles(&cclfMap, &skipped))
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
}

func (s *CCLFTestSuite) TestOrderACOs() {
	origACOs := os.Getenv("CCLF_PRIORITY_ACO_CMS_IDS")
	os.Setenv("CCLF_PRIORITY_ACO_CMS_IDS", "A3456, A8765, A4321")
	defer os.Setenv("CCLF_PRIORITY_ACO_CMS_IDS", origACOs)

	var cclfMap = map[string]map[int][]*cclfFileMetadata{
		"A1111": map[int][]*cclfFileMetadata{},
		"A8765": map[int][]*cclfFileMetadata{},
		"A3456": map[int][]*cclfFileMetadata{},
		"A0246": map[int][]*cclfFileMetadata{},
	}

	acoOrder := orderACOs(&cclfMap)

	assert.Len(s.T(), acoOrder, 4)
	assert.Equal(s.T(), "A3456", acoOrder[0])
	assert.Equal(s.T(), "A8765", acoOrder[1])
	assert.Regexp(s.T(), "A1111|A0246", acoOrder[2])
	assert.Regexp(s.T(), "A1111|A0246", acoOrder[3])
}

func (s *CCLFTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[int][]*cclfFileMetadata)
	testUtils.SetPendingDeletionDir(s.Suite)

	// failed import: file that's within the threshold - stay put
	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf0metadata := &cclfFileMetadata{
		name:         "T.A0001.ACO.ZC0Y18.D181120.T1000011",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC0Y18.D181120.T1000011",
		imported:     false,
		deliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	acoID = "A0001"
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf8metadata := &cclfFileMetadata{
		name:         "T.A0001.ACO.ZC8Y18.D181120.T1000009",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		filePath:     BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009",
		imported:     false,
		deliveryDate: fileTime,
	}

	// successfully imported file - should move
	acoID = "A0001"
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf9metadata := &cclfFileMetadata{
		name:      "T.A0001.ACO.ZC9Y18.D181120.T1000010",
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		imported:  true,
	}
	cclfmap["A0001"] = map[int][]*cclfFileMetadata{18: []*cclfFileMetadata{cclf0metadata, cclf8metadata, cclf9metadata}}
	err := cleanUpCCLF(cclfmap)
	assert.Nil(err)

	files, err := ioutil.ReadDir(os.Getenv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", os.Getenv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual("T.A0001.ACO.ZC0Y18.D181120.T1000011", file.Name())
	}
	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf/")
}

func deleteFilesByACO(acoID string, db *gorm.DB) error {
	var files []models.CCLFFile
	db.Where("aco_cms_id = ?", acoID).Find(&files)
	for _, cclfFile := range files {
		err := cclfFile.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}
