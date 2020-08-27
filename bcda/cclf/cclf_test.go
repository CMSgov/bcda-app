package cclf

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

const BASE_FILE_PATH = "../../shared_files/"

var origDate string

type CCLFTestSuite struct {
	suite.Suite
	pendingDeletionDir string
}

func (s *CCLFTestSuite) SetupTest() {
	models.InitializeGormModels()
	os.Setenv("CCLF_REF_DATE", "181201")
}

func (s *CCLFTestSuite) SetupSuite() {
	origDate = os.Getenv("CCLF_REF_DATE")

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)
}

func (s *CCLFTestSuite) TearDownSuite() {
	os.Setenv("CCLF_REF_DATE", origDate)
	os.RemoveAll(s.pendingDeletionDir)
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

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var fs []models.CCLFFile
	db.Where("aco_cms_id in (?, ?, ?)", aco1, aco2, aco3).Find(&fs)
	for _, f := range fs {
		err := f.Delete()
		assert.Nil(err)
	}

	sc, f, sk, err := ImportCCLFDirectory(BASE_FILE_PATH + "cclf/archives/valid/")
	assert.Nil(err)
	assert.Equal(6, sc)
	assert.Equal(0, f)
	assert.Equal(1, sk)

	var aco1fs, aco2fs, aco3fs []models.CCLFFile
	db.Where("aco_cms_id = ?", aco1).Find(&aco1fs)
	db.Where("aco_cms_id = ?", aco2).Find(&aco2fs)
	db.Where("aco_cms_id = ?", aco3).Find(&aco3fs)

	assert.True(aco1fs[0].CreatedAt.Before(aco2fs[0].CreatedAt))
	assert.True(aco2fs[0].CreatedAt.Before(aco3fs[0].CreatedAt))

	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf/archives/valid/")
}

func (s *CCLFTestSuite) TestImportCCLF0() {
	ctx := context.Background()
	assert := assert.New(s.T())

	cclf0filePath := BASE_FILE_PATH + "cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000"
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011"}

	// positive
	validator, err := importCCLF0(ctx, cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])

	// negative
	cclf0metadata = &cclfFileMetadata{}
	_, err = importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "could not read CCLF0 archive : read .: is a directory")

	// missing cclf8 from cclf0
	cclf0filePath = BASE_FILE_PATH + "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181120.T1000000"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011"}
	_, err = importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "failed to parse CCLF8 from CCLF0 file T.BCD.A0001.ZC0Y18.D181120.T1000011")

	// duplicate file types from cclf0
	cclf0filePath = BASE_FILE_PATH + "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181122.T1000000"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000013"}
	_, err = importCCLF0(ctx, cclf0metadata)
	assert.EqualError(err, "duplicate CCLF8 file type found from CCLF0 file")
}

func (s *CCLFTestSuite) TestImportCCLF0_SplitFiles() {
	assert := assert.New(s.T())

	cclf0filePath := BASE_FILE_PATH + "cclf/archives/split/T.BCD.A0001.ZCY18.D181120.T1000000"
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18, name: "T.BCD.A0001.ZC0Y18.D181120.T1000011-1"}

	validator, err := importCCLF0(context.Background(), cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])
}

func (s *CCLFTestSuite) TestValidate() {
	ctx := context.Background()
	assert := assert.New(s.T())

	cclf8filePath := BASE_FILE_PATH + "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000"
	cclf8metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18, name: "T.BCD.A0001.ZC8Y18.D181120.T1000009"}

	// positive
	cclfvalidator := map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 6, maxRecordLength: 549}}
	err := validate(ctx, cclf8metadata, cclfvalidator)
	assert.Nil(err)

	// negative
	cclfvalidator = map[string]cclfFileValidator{"CCLF8": {totalRecordCount: 2, maxRecordLength: 549}}
	err = validate(ctx, cclf8metadata, cclfvalidator)
	assert.EqualError(err, "maximum record count reached for file CCLF8 (expected: 2, actual: 3)")
}

func (s *CCLFTestSuite) TestValidate_FolderName() {
	assert := assert.New(s.T())

	filePath := BASE_FILE_PATH + "path/T.BCD.A0001.ZCY18.D181120.T1000000"
	err := validateCCLFFolderName(filePath)
	assert.Nil(err)

	filePath = BASE_FILE_PATH + "path/T.A0001.ACO.ZC8Y18.D18NOV20.T1000009"
	err = validateCCLFFolderName(filePath)
	assert.EqualError(err, fmt.Sprintf("invalid foldername for CCLF archive: %s", filePath))

	filePath = BASE_FILE_PATH + "path/T.BCD.ACO.ZC0Y18.D181120.T0001000"
	err = validateCCLFFolderName(filePath)
	assert.EqualError(err, fmt.Sprintf("invalid foldername for CCLF archive: %s", filePath))
}

func (s *CCLFTestSuite) TestParseTimestamp() {
	assert := assert.New(s.T())

	cclfMetadata, err := getCCLFFileMetadata("T.BCD.A0001.ZC8Y18.D181120.T1000009")
	assert.Nil(err)
	assert.Equal(10, cclfMetadata.timestamp.Hour())
	assert.Equal(00, cclfMetadata.timestamp.Minute())

	// valid file name out of range
	cclfMetadata, err = getCCLFFileMetadata("T.BCD.A0001.ZC8Y18.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: T.BCD.A0001.ZC8Y18.D190117.T9909420: parsing time \"D190117.T990942\": hour out of range")
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
		name:      "T.BCD.A0001.ZC8Y18.D181120.T1000009",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  BASE_FILE_PATH + "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000",
	}

	err = importCCLF8(context.Background(), metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := models.CCLFFile{}
	db.First(&file, "name = ?", metadata.name)
	assert.NotNil(file)
	assert.Equal("T.BCD.A0001.ZC8Y18.D181120.T1000009", file.Name)
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
	err := importCCLF8(context.Background(), metadata)
	assert.EqualError(err, "CCLF file not found")
}

func (s *CCLFTestSuite) TestGetCCLFFileMetadata() {
	assert := assert.New(s.T())

	expDate, _ := time.Parse("2006-01-02", "2019-01-19")
	os.Setenv("CCLF_REF_DATE", "190120")
	metadata, err := getCCLFFileMetadata("T.BCD.A0002.ZC0Y18.D190119.T1000009")
	assert.Equal("test", metadata.env)
	assert.Equal("A0002", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(expDate.Year(), metadata.timestamp.Year()) // comparing only date - with new format, time stores ACO ID
	assert.Equal(expDate.Month(), metadata.timestamp.Month())
	assert.Equal(expDate.Day(), metadata.timestamp.Day())
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	expDate, _ = time.Parse("2006-01-02", "2019-01-17")
	metadata, err = getCCLFFileMetadata("T.BCD.A0000.ZC8Y18.D190117.T0000000")
	assert.Equal("test", metadata.env)
	assert.Equal("A0000", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(expDate.Year(), metadata.timestamp.Year())
	assert.Equal(expDate.Month(), metadata.timestamp.Month())
	assert.Equal(expDate.Day(), metadata.timestamp.Day())
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	expDate, _ = time.Parse("2006-01-02", "2019-01-08")
	metadata, err = getCCLFFileMetadata("/path/P.A0001.ACO.ZC9Y18.D190108.T2355000")
	assert.EqualError(err, "invalid filename for file: /path/P.A0001.ACO.ZC9Y18.D190108.T2355000")

	// CMS EFT file format with BCD identifier
	expDate, _ = time.Parse("2006-01-02", "2018-11-20")
	os.Setenv("CCLF_REF_DATE", "181124")
	metadata, err = getCCLFFileMetadata("T.BCD.A0001.ZC0Y18.D181120.T0001000")
	assert.Equal("test", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(expDate.Year(), metadata.timestamp.Year())
	assert.Equal(expDate.Month(), metadata.timestamp.Month())
	assert.Equal(expDate.Day(), metadata.timestamp.Day())
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	//CMS EFT file format with ACOB identifier
	expDate, _ = time.Parse("2006-01-02", "2018-11-20")
	metadata, err = getCCLFFileMetadata("/BCD/T.BCD.ACOB.ZC0Y18.D181120.T0001000")
	assert.EqualError(err, "invalid filename for file: /BCD/T.BCD.ACOB.ZC0Y18.D181120.T0001000")

	expDate, _ = time.Parse("2006-01-02", "2019-01-12")
	os.Setenv("CCLF_REF_DATE", "190120")
	metadata, err = getCCLFFileMetadata("T.BCD.A0012.ZC8Y18.D190112.T1000009")
	assert.Equal("test", metadata.env)
	assert.Equal("A0012", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(expDate.Year(), metadata.timestamp.Year())
	assert.Equal(expDate.Month(), metadata.timestamp.Month())
	assert.Equal(expDate.Day(), metadata.timestamp.Day())
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	os.Setenv("CCLF_REF_DATE", "180612")
	metadata, err = getCCLFFileMetadata("/BCD/P.BCD.ACO.ZC9Y19.D180610.T0002000")
	assert.EqualError(err, "invalid filename for file: /BCD/P.BCD.ACO.ZC9Y19.D180610.T0002000")

	os.Unsetenv("CCLF_REF_DATE")
}

func (s *CCLFTestSuite) TestGetCCLFFileMetadata_DateOutOfRange() {
	assert := assert.New(s.T())

	// File is postdated
	os.Setenv("CCLF_REF_DATE", "181119")
	_, err := getCCLFFileMetadata("T.BCD.A0001.ZC0Y18.D181120.T0001000")
	assert.EqualError(err, "date 'D181120.T000100' from file T.BCD.A0001.ZC0Y18.D181120.T0001000 out of range; comparison date 181119")

	// File is older than 45 days
	os.Setenv("CCLF_REF_DATE", "190120")
	_, err = getCCLFFileMetadata("T.BCD.A0001.ZC0Y18.D181120.T0001000")
	assert.EqualError(err, "date 'D181120.T000100' from file T.BCD.A0001.ZC0Y18.D181120.T0001000 out of range; comparison date 190120")
}

func (s *CCLFTestSuite) TestGetCCLFFileMetadata_InvalidFilename() {
	assert := assert.New(s.T())
	os.Setenv("CCLF_REF_DATE", "190615")

	_, err := getCCLFFileMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	_, err = getCCLFFileMetadata("T.BCD.A0001.ZC8Y18.D191317.T0000000")
	assert.EqualError(err, "failed to parse date 'D191317.T000000' from file: T.BCD.A0001.ZC8Y18.D191317.T0000000: parsing time \"D191317.T000000\": month out of range")

	_, err = getCCLFFileMetadata("/cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")
	assert.EqualError(err, "invalid filename for file: /cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")

	_, err = getCCLFFileMetadata("/cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
	assert.EqualError(err, "invalid filename for file: /cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
}

func (s *CCLFTestSuite) TestSortCCLFArchives() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[int][]*cclfFileMetadata)
	var skipped int

	filePath := BASE_FILE_PATH + "cclf/archives/valid/"
	err := filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(cclfmap["A0001"][18]))
	assert.Equal(1, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf/archives/bcd/"
	err = filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(cclfmap["A0001"][18]))
	assert.Equal(1, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf/mixed/with_invalid_filenames/"
	err = filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist := cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist))
	assert.Equal(4, skipped)
	for _, cclf := range cclflist {
		assert.NotEqual(9, cclf.cclfNum)
	}
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf/mixed/0/valid_names/"
	err = filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(3, len(cclflist)) // 3 archives
	assert.Equal(3, skipped)       // 3 files
	for _, cclf := range cclflist {
		assert.Equal(0, cclf.cclfNum)
	}

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf/archives/8/valid/"
	err = filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(5, len(cclflist))
	assert.Equal(0, skipped)
	for _, cclf := range cclflist {
		assert.Equal(8, cclf.cclfNum)
	}

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf/files/9/valid_names/"
	err = filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(0, len(cclflist))
	assert.Equal(4, skipped)
	testUtils.ResetFiles(s.Suite, filePath)

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	filePath = BASE_FILE_PATH + "cclf/mixed/with_folders/"
	err = filepath.Walk(filePath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist)) // 2 archives
	assert.Equal(13, skipped)      // 13 files
	var cclf0, cclf8 []*cclfFileMetadata

	for _, cclf := range cclflist {
		if cclf.cclfNum == 0 {
			cclf0 = append(cclf0, cclf)
		} else if cclf.cclfNum == 8 {
			cclf8 = append(cclf8, cclf)
		}
	}
	assert.Equal(1, len(cclf8))
	assert.Equal(1, len(cclf0))
	testUtils.ResetFiles(s.Suite, filePath)
}

func (s *CCLFTestSuite) TestSortCCLFArchives_TimeChange() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[int][]*cclfFileMetadata)
	var skipped int
	folderPath := BASE_FILE_PATH + "cclf/mixed/with_invalid_filenames/"
	filePath := folderPath + "T.BCDE.ACO.ZC0Y18.D181120.T0001000"

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err.Error())
	}

	skipped = 0
	err = filepath.Walk(folderPath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist := cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist))
	assert.Equal(4, skipped)
	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf/mixed/with_invalid_filenames/")

	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	cclfmap = make(map[string]map[int][]*cclfFileMetadata)
	skipped = 0
	err = filepath.Walk(folderPath, sortCCLFArchives(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001"][18]
	assert.Equal(2, len(cclflist))
	assert.Equal(4, skipped)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, "open ../../shared_files/cclf/mixed/with_invalid_filenames/T.BCDE.ACO.ZC0Y18.D181120.T0001000: no such file or directory")

	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf/mixed/with_invalid_filenames/")
}

func (s *CCLFTestSuite) TestSortCCLFArchives_InvalidPath() {
	cclfMap := make(map[string]map[int][]*cclfFileMetadata)
	skipped := 0
	err := filepath.Walk("./foo", sortCCLFArchives(&cclfMap, &skipped))
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
}

func (s *CCLFTestSuite) TestOrderACOs() {
	var cclfMap = map[string]map[int][]*cclfFileMetadata{
		"A1111": map[int][]*cclfFileMetadata{},
		"A8765": map[int][]*cclfFileMetadata{},
		"A3456": map[int][]*cclfFileMetadata{},
		"A0246": map[int][]*cclfFileMetadata{},
	}

	acoOrder := orderACOs(&cclfMap)

	// A3456 and A8765 have been added to the database == prioritized over the other two
	assert.Len(s.T(), acoOrder, 4)
	assert.Equal(s.T(), "A3456", acoOrder[0])
	assert.Equal(s.T(), "A8765", acoOrder[1])
	assert.Regexp(s.T(), "A1111|A0246", acoOrder[2])
	assert.Regexp(s.T(), "A1111|A0246", acoOrder[3])
}

func (s *CCLFTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[int][]*cclfFileMetadata)

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
		filePath:     BASE_FILE_PATH + "cclf/T.BCD.ACO.ZC0Y18.D181120.T0001000",
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
		filePath:     BASE_FILE_PATH + "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000",
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
		filePath:  BASE_FILE_PATH + "cclf/archives/valid/T.BCD.A0001.ZCY18.D181122.T1000000",
		imported:  true,
	}
	cclfmap["A0001"] = map[int][]*cclfFileMetadata{18: []*cclfFileMetadata{cclf0metadata, cclf8metadata, cclf9metadata}}
	err := cleanUpCCLF(context.Background(), cclfmap)
	assert.Nil(err)

	files, err := ioutil.ReadDir(os.Getenv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", os.Getenv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual("T.BCD.ACO.ZC0Y18.D181120.T0001000", file.Name())
	}
	testUtils.ResetFiles(s.Suite, BASE_FILE_PATH+"cclf/archives/valid/")
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
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			gdb, err := gorm.Open("postgres", db)
			if err != nil {
				t.Fatalf("Failed to instantiate gorm db %s", err.Error())
			}
			gdb.LogMode(true)
			defer func() {
				assert.NoError(t, mock.ExpectationsWereMet())
				gdb.Close()
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

			result := getPriorityACOs(gdb)
			if tt.errToReturn != nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.idsToReturn, result)
			}
		})
	}
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
