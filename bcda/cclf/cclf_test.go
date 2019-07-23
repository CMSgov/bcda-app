package cclf

import (
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 54}, validator["CCLF9"])

	// negative
	cclf0metadata = &cclfFileMetadata{}
	_, err = importCCLF0(cclf0metadata)
	assert.NotNil(err)
	assert.EqualError(err, "could not read CCLF0 archive : read .: is a directory")

	// missing cclf8 and or 9 from cclf0
	cclf0filePath = BASE_FILE_PATH + "cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "failed to parse CCLF8 from CCLF0 file")

	cclf0filePath = BASE_FILE_PATH + "cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000012"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "failed to parse CCLF9 from CCLF0 file")

	// duplicate file types from cclf0
	cclf0filePath = BASE_FILE_PATH + "cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000013"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "duplicate CCLF9 file type found from CCLF0 file.")
}

func (s *CCLFTestSuite) TestImportCCLF0_SplitFiles() {
	assert := assert.New(s.T())

	cclf0filePath := BASE_FILE_PATH + "cclf_split/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}

	validator, err := importCCLF0(cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 54}, validator["CCLF9"])
}

func (s *CCLFTestSuite) TestValidate() {
	assert := assert.New(s.T())

	cclf8filePath := BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"
	cclf8metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18}

	cclf9filePath := BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010"
	cclf9metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 9, timestamp: time.Now(), filePath: cclf9filePath, perfYear: 18}

	// positive
	cclfvalidator := map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 6, maxRecordLength: 549}, "CCLF9": {totalRecordCount: 6, maxRecordLength: 54}}
	err := validate(cclf8metadata, cclfvalidator)
	assert.Nil(err)
	err = validate(cclf9metadata, cclfvalidator)
	assert.Nil(err)

	// negative
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 2, maxRecordLength: 549}, "CCLF9": {totalRecordCount: 6, maxRecordLength: 3}}
	err = validate(cclf8metadata, cclfvalidator)
	assert.NotNil(err)
	err = validate(cclf9metadata, cclfvalidator)
	assert.NotNil(err)
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

	cclf9Metadata := &cclfFileMetadata{
		acoID:     "A0001",
		cclfNum:   8,
		timestamp: time.Now(),
		filePath:  BASE_FILE_PATH + "cclf_split/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		perfYear:  18,
	}

	validator := map[string]cclfFileValidator{
		"CCLF8": {totalRecordCount: 6, maxRecordLength: 549},
		"CCLF9": {totalRecordCount: 6, maxRecordLength: 54},
	}

	err := validate(cclf8Metadata, validator)
	assert.Nil(err)

	err = validate(cclf9Metadata, validator)
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
	assert.NotNil(err)
	assert.EqualError(err, "CCLF file not found")
}

func (s *CCLFTestSuite) TestImportCCLF9() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err := deleteFilesByACO("A0001", db)
	assert.Nil(err)

	acoID := "A0002"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf9metadata := &cclfFileMetadata{
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		timestamp: fileTime,
		filePath:  BASE_FILE_PATH + "cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		perfYear:  18,
		name:      "T.A0001.ACO.ZC9Y18.D181120.T1000010",
	}

	err = importCCLF9(cclf9metadata)
	assert.Nil(err)

	file := models.CCLFFile{}
	db.Where("name = ?", cclf9metadata.name).Last(&file)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC9Y18.D181120.T1000010", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime.Format("010203040506"), file.Timestamp.Format("010203040506"))
	assert.Equal(18, file.PerformanceYear)

	var savedCCLF9 models.CCLFBeneficiaryXref
	db.Where("current_num = ? and file_id = ?", "1A69B98CD35", file.ID).Last(&savedCCLF9)
	assert.NotNil(savedCCLF9)
	assert.Equal("M", savedCCLF9.XrefIndicator)
	assert.Equal("1A69B98CD35", savedCCLF9.CurrentNum)
	assert.Equal("1A69B98CD34", savedCCLF9.PrevNum)
	assert.Equal("1960-01-01", savedCCLF9.PrevsEfctDt)
	assert.Equal("2010-05-11", savedCCLF9.PrevsObsltDt)

	err = deleteFilesByACO("A0001", db)
	assert.Nil(err)
}

func (s *CCLFTestSuite) TestImportCCLF9_SplitFiles() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err := deleteFilesByACO("A0001", db)
	assert.Nil(err)

	acoID := "A0002"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf9metadata := &cclfFileMetadata{
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		timestamp: fileTime,
		filePath:  BASE_FILE_PATH + "cclf_split/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		perfYear:  18,
		name:      "T.A0001.ACO.ZC9Y18.D181120.T1000010",
	}

	err = importCCLF9(cclf9metadata)
	assert.Nil(err)

	file := models.CCLFFile{}
	db.Where("name = ?", cclf9metadata.name).Last(&file)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC9Y18.D181120.T1000010", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime.Format("010203040506"), file.Timestamp.Format("010203040506"))
	assert.Equal(18, file.PerformanceYear)

	var savedCCLF9 models.CCLFBeneficiaryXref
	db.Find(&savedCCLF9, "file_id = ?", &file.ID)
	assert.NotNil(savedCCLF9)
	assert.Equal("M", savedCCLF9.XrefIndicator)
	assert.Equal("1A69B98CD35", savedCCLF9.CurrentNum)
	assert.Equal("1A69B98CD34", savedCCLF9.PrevNum)
	assert.Equal("1960-01-01", savedCCLF9.PrevsEfctDt)
	assert.Equal("2010-05-11", savedCCLF9.PrevsObsltDt)

	err = deleteFilesByACO("A0001", db)
	assert.Nil(err)
}

func (s *CCLFTestSuite) TestImportCCLF9_InvalidMetadata() {
	assert := assert.New(s.T())

	var metadata *cclfFileMetadata
	err := importCCLF9(metadata)
	assert.NotNil(err)
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
	assert.Equal("production", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(9, metadata.cclfNum)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	// CMS EFT file format structure
	expTime, _ = time.Parse(time.RFC3339, "2019-01-19T20:13:01Z")
	metadata, err = getCCLFFileMetadata("/cclf/T#EFT.ON.A0001.ACOB.ZC0Y19.D190119.T2013010")
	assert.Equal("test", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Equal(19, metadata.perfYear)
	assert.Nil(err)
}

func (s *CCLFTestSuite) TestGetCCLFFileMetadata_InvalidFilename() {
	assert := assert.New(s.T())

	_, err := getCCLFFileMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	_, err = getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: /path/T.A0000.ACO.ZC8Y18.D190117.T9909420: parsing time \"D190117.T990942\": hour out of range")

	_, err = getCCLFFileMetadata("/cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")
	assert.NotNil(err)

	_, err = getCCLFFileMetadata("/cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
	assert.NotNil(err)
}

func (s *CCLFTestSuite) TestSortCCLFFiles() {
	assert := assert.New(s.T())
	cclfmap := make(map[string][]*cclfFileMetadata)
	var skipped int

	filePath := BASE_FILE_PATH + "cclf/"
	err := filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	assert.Equal(3, len(cclfmap["A0001_18"]))
	assert.Equal(0, skipped)

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf_BadFileNames/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist := cclfmap["A0001_18"]
	assert.Equal(2, len(cclflist))
	assert.Equal(2, skipped)
	for _, cclf := range cclflist {
		assert.NotEqual(9, cclf.cclfNum)
	}
	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf_BadFileNames/")

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf0/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	assert.Equal(3, len(cclflist))
	assert.Equal(0, skipped)
	for _, cclf := range cclflist {
		assert.Equal(0, cclf.cclfNum)
	}

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf8/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	assert.Equal(5, len(cclflist))
	assert.Equal(0, skipped)
	for _, cclf := range cclflist {
		assert.Equal(8, cclf.cclfNum)
	}

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf9/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	assert.Equal(4, len(cclflist))
	assert.Equal(0, skipped)
	modtimeBefore := cclflist[0].deliveryDate
	modtimeAfter := time.Now().Truncate(time.Second)
	for _, cclf := range cclflist {
		assert.Equal(9, cclf.cclfNum)
		assert.Equal(modtimeBefore.Format("010203040506"), cclf.deliveryDate.Format("010203040506"))

		// change the modification time for all the files
		err := os.Chtimes(cclf.filePath, modtimeAfter, modtimeAfter)
		if err != nil {
			s.FailNow("Failed to change modified time for file", err)
		}
	}

	cclfmap = make(map[string][]*cclfFileMetadata)
	filePath = BASE_FILE_PATH + "cclf9/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	for _, cclf := range cclflist {
		// check for the new modification time
		assert.Equal(modtimeAfter.Format("010203040506"), cclf.deliveryDate.Format("010203040506"))
	}

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf_All/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	assert.Equal(12, len(cclflist))
	assert.Equal(0, skipped)
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
	assert.Equal(4, len(cclf9))
	assert.Equal(3, len(cclf0))
}

func (s *CCLFTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string][]*cclfFileMetadata)
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
	cclfmap["A0001_18"] = []*cclfFileMetadata{cclf0metadata, cclf8metadata, cclf9metadata}
	err := cleanupCCLF(cclfmap)
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

func (s *CCLFTestSuite) TestIsSuppressionFile() {
	suppressionFilePath := BASE_FILE_PATH + "cclf_Suppression/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"
	assert.True(s.T(),isSuppressionFile(suppressionFilePath))

	suppressionFilePath = BASE_FILE_PATH + "cclf_Suppression/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	assert.False(s.T(),isSuppressionFile(suppressionFilePath))
}

func (s *CCLFTestSuite) TestDeleteDirectory() {
	assert := assert.New(s.T())
	dirToDelete := BASE_FILE_PATH + "doomedDirectory"
	testUtils.MakeDirToDelete(s.Suite, dirToDelete)
	defer os.Remove(dirToDelete)

	f, err := os.Open(dirToDelete)
	assert.Nil(err)
	files, err := f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(4, len(files))

	filesDeleted, err := DeleteDirectoryContents(dirToDelete)
	assert.Equal(4, filesDeleted)
	assert.Nil(err)

	f, err = os.Open(dirToDelete)
	assert.Nil(err)
	files, err = f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(0, len(files))

	filesDeleted, err = DeleteDirectoryContents("This/Does/not/Exist")
	assert.Equal(0, filesDeleted)
	assert.NotNil(err)
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
