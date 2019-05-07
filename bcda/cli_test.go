package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type CLITestSuite struct {
	suite.Suite
	testApp *cli.App
}

func (s *CLITestSuite) SetupTest() {
	s.testApp = setUpApp()
	models.InitializeGormModels()
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestCreateACO() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	// Successful ACO creation
	ACOName := "Unit Test ACO 1"
	args := []string{"bcda", "create-aco", "--name", ACOName}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID := strings.TrimSpace(buf.String())
	var testACO models.ACO
	db.First(&testACO, "Name=?", ACOName)
	assert.Equal(testACO.UUID.String(), acoUUID)
	buf.Reset()

	ACO2Name := "Unit Test ACO 2"
	aco2ID := "A9999"
	args = []string{"bcda", "create-aco", "--name", ACO2Name, "--cms-id", aco2ID}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID = strings.TrimSpace(buf.String())
	var testACO2 models.ACO
	db.First(&testACO2, "Name=?", ACO2Name)
	assert.Equal(testACO2.UUID.String(), acoUUID)
	assert.Equal(*testACO2.CMSID, aco2ID)
	buf.Reset()

	// Negative tests

	// No parameters
	args = []string{"bcda", "create-aco"}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// No ACO Name
	badACO := ""
	args = []string{"bcda", "create-aco", "--name", badACO}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// ACO name without flag
	args = []string{"bcda", "create-aco", ACOName}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Unexpected flag
	args = []string{"bcda", "create-aco", "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// Invalid CMS ID
	args = []string{"bcda", "create-aco", "--name", ACOName, "--cms-id", "ABCDE"}
	err = s.testApp.Run(args)
	assert.Equal("ACO CMS ID (--cms-id) is invalid", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()
}

func (s *CLITestSuite) TestImportCCLF0() {
	assert := assert.New(s.T())

	cclf0filePath := "../shared_files/cclf/T.A0001.ACO.ZC0Y18.D181120.T1000011"
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

	// missing cclf8 and or 9 from cclf0
	cclf0filePath = "../shared_files/cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.NotNil(err)

	cclf0filePath = "../shared_files/cclf0_MissingData/T.A0001.ACO.ZC0Y18.D181120.T1000012"
	cclf0metadata = &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}
	_, err = importCCLF0(cclf0metadata)
	assert.NotNil(err)
}

func (s *CLITestSuite) TestImportCCLF0_SplitFiles() {
	assert := assert.New(s.T())

	cclf0filePath := "../shared_files/cclf_split/T.A0001.ACO.ZC0Y18.D181120.T1000011"
	cclf0metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 0, timestamp: time.Now(), filePath: cclf0filePath, perfYear: 18}

	validator, err := importCCLF0(cclf0metadata)
	assert.Nil(err)
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 549}, validator["CCLF8"])
	assert.Equal(cclfFileValidator{totalRecordCount: 6, maxRecordLength: 54}, validator["CCLF9"])
}

func (s *CLITestSuite) TestValidate() {
	assert := assert.New(s.T())

	cclf8filePath := "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"
	cclf8metadata := &cclfFileMetadata{env: "test", acoID: "A0001", cclfNum: 8, timestamp: time.Now(), filePath: cclf8filePath, perfYear: 18}

	cclf9filePath := "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010"
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

func (s *CLITestSuite) TestValidate_SplitFiles() {
	assert := assert.New(s.T())

	cclf8Metadata := &cclfFileMetadata{
		acoID:     "A0001",
		cclfNum:   8,
		timestamp: time.Now(),
		filePath:  "../shared_files/cclf_split/T.A0001.ACO.ZC8Y18.D181120.T1000009",
		perfYear:  18,
	}

	cclf9Metadata := &cclfFileMetadata{
		acoID:     "A0001",
		cclfNum:   8,
		timestamp: time.Now(),
		filePath:  "../shared_files/cclf_split/T.A0001.ACO.ZC9Y18.D181120.T1000010",
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

func (s *CLITestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	//db.Unscoped().Delete(&models.CCLFBeneficiary{})
	//db.Unscoped().Delete(&models.CCLFFile{})

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &cclfFileMetadata{
		name:      "T.A0001.ACO.ZC8Y18.D181120.T1000009",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009",
	}

	err := importCCLF8(metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := models.CCLFFile{}
	db.First(&file, "name = ?", metadata.name)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC8Y18.D181120.T1000009", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime, file.Timestamp)
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

	//db.Unscoped().Delete(&models.CCLFBeneficiary{})
	//db.Unscoped().Delete(&models.CCLFFile{})
}

func (s *CLITestSuite) TestImportCCLF8_SplitFiles() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	//db.Unscoped().Delete(&models.CCLFBeneficiary{})
	//db.Unscoped().Delete(&models.CCLFFile{})

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &cclfFileMetadata{
		name:      "T.A0001.ACO.ZC8Y18.D181120.T1000009",
		env:       "test",
		acoID:     acoID,
		cclfNum:   8,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  "../shared_files/cclf_split/T.A0001.ACO.ZC8Y18.D181120.T1000009",
	}

	err := importCCLF8(metadata)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	file := models.CCLFFile{}
	db.First(&file, "name = ?", metadata.name)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC8Y18.D181120.T1000009", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime, file.Timestamp)
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

	//db.Unscoped().Delete(&models.CCLFBeneficiary{})
	//db.Unscoped().Delete(&models.CCLFFile{})
}

func (s *CLITestSuite) TestImportCCLF8_InvalidMetadata() {
	assert := assert.New(s.T())

	var metadata *cclfFileMetadata
	err := importCCLF8(metadata)
	assert.NotNil(err)
	assert.EqualError(err, "CCLF file not found")
}

func (s *CLITestSuite) TestImportCCLF9() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	//db.Unscoped().Delete(&models.CCLFBeneficiaryXref{})
	//db.Unscoped().Delete(&models.CCLFFile{})

	acoID := "A0002"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf9metadata := &cclfFileMetadata{
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		timestamp: fileTime,
		filePath:  "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		perfYear:  18,
		name:      "T.A0001.ACO.ZC9Y18.D181120.T1000010",
	}

	err := importCCLF9(cclf9metadata)
	assert.Nil(err)

	file := models.CCLFFile{}
	db.First(&file, "name = ?", cclf9metadata.name)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC9Y18.D181120.T1000010", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime, file.Timestamp)
	assert.Equal(18, file.PerformanceYear)

	var savedCCLF9 models.CCLFBeneficiaryXref
	db.Find(&savedCCLF9, "id = ?", "6")
	assert.NotNil(savedCCLF9)
	assert.Equal("M", savedCCLF9.XrefIndicator)
	assert.Equal("1A69B98CD35", savedCCLF9.CurrentNum)
	assert.Equal("1A69B98CD34", savedCCLF9.PrevNum)
	assert.Equal("1960-01-01", savedCCLF9.PrevsEfctDt)
	assert.Equal("2010-05-11", savedCCLF9.PrevsObsltDt)

	//db.Unscoped().Delete(&models.CCLFBeneficiaryXref{})
	//db.Unscoped().Delete(&models.CCLFFile{})
}

func (s *CLITestSuite) TestImportCCLF9_SplitFiles() {
	assert := assert.New(s.T())
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	//db.Unscoped().Delete(&models.CCLFBeneficiaryXref{})
	//db.Unscoped().Delete(&models.CCLFFile{})

	acoID := "A0002"
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	cclf9metadata := &cclfFileMetadata{
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		timestamp: fileTime,
		filePath:  "../shared_files/cclf_split/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		perfYear:  18,
		name:      "T.A0001.ACO.ZC9Y18.D181120.T1000010",
	}

	err := importCCLF9(cclf9metadata)
	assert.Nil(err)

	file := models.CCLFFile{}
	db.First(&file, "name = ?", cclf9metadata.name)
	assert.NotNil(file)
	assert.Equal("T.A0001.ACO.ZC9Y18.D181120.T1000010", file.Name)
	assert.Equal(acoID, file.ACOCMSID)
	assert.Equal(fileTime, file.Timestamp)
	assert.Equal(18, file.PerformanceYear)

	var savedCCLF9 models.CCLFBeneficiaryXref
	db.Find(&savedCCLF9, "file_id = ?", &file.ID)
	assert.NotNil(savedCCLF9)
	assert.Equal("M", savedCCLF9.XrefIndicator)
	assert.Equal("1A69B98CD35", savedCCLF9.CurrentNum)
	assert.Equal("1A69B98CD34", savedCCLF9.PrevNum)
	assert.Equal("1960-01-01", savedCCLF9.PrevsEfctDt)
	assert.Equal("2010-05-11", savedCCLF9.PrevsObsltDt)

	//db.Unscoped().Delete(&models.CCLFBeneficiaryXref{})
	//db.Unscoped().Delete(&models.CCLFFile{})
}

func (s *CLITestSuite) TestImportCCLF9_InvalidMetadata() {
	assert := assert.New(s.T())

	var metadata *cclfFileMetadata
	err := importCCLF9(metadata)
	assert.NotNil(err)
	assert.EqualError(err, "CCLF file not found")
}

func (s *CLITestSuite) TestGetCCLFFileMetadata() {
	assert := assert.New(s.T())

	_, err := getCCLFFileMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	metadata, err := getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: /path/T.A0000.ACO.ZC8Y18.D190117.T9909420")

	expTime, _ := time.Parse(time.RFC3339, "2019-01-17T21:09:42Z")
	metadata, err = getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T2109420")
	assert.Equal("test", metadata.env)
	assert.Equal("A0000", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	expTime, _ = time.Parse(time.RFC3339, "2019-01-08T23:55:00Z")
	metadata, err = getCCLFFileMetadata("/path/P.A0001.ACO.ZC9Y18.D190108.T2355000")
	assert.Equal("production", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(9, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	expTime, _ = time.Parse(time.RFC3339, "2019-01-19T20:13:01Z")
	metadata, err = getCCLFFileMetadata("/path/T.A0002.ACO.ZC0Y18.D190119.T2013010")
	assert.Equal("test", metadata.env)
	assert.Equal("A0002", metadata.acoID)
	assert.Equal(0, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Equal(18, metadata.perfYear)
	assert.Nil(err)

	metadata, err = getCCLFFileMetadata("/cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010")
	assert.NotNil(err)

	metadata, err = getCCLFFileMetadata("/cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010")
	assert.NotNil(err)
}

func (s *CLITestSuite) TestSortCCLFFiles() {
	assert := assert.New(s.T())
	cclfmap := make(map[string][]*cclfFileMetadata)
	var skipped int

	filePath := "../shared_files/cclf/"
	err := filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	assert.Equal(3, len(cclfmap["A0001_18"]))
	assert.Equal(0, skipped)

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = "../shared_files/cclf_BadFileNames/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist := cclfmap["A0001_18"]
	assert.Equal(2, len(cclflist))
	assert.Equal(2, skipped)
	for _, cclf := range cclflist {
		assert.NotEqual(s.T(), 9, cclf.cclfNum)
	}
	s.resetFiles("../shared_files/cclf_BadFileNames/")

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = "../shared_files/cclf0/"
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
	filePath = "../shared_files/cclf8/"
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
	filePath = "../shared_files/cclf9/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	assert.Equal(4, len(cclflist))
	assert.Equal(0, skipped)
	modtimeBefore := cclflist[0].deliveryDate
	modtimeAfter := time.Now().Truncate(time.Second)
	for _, cclf := range cclflist {
		assert.Equal(9, cclf.cclfNum)
		assert.Equal(modtimeBefore, cclf.deliveryDate)

		// change the modification time for all the files
		err := os.Chtimes(cclf.filePath, modtimeAfter, modtimeAfter)
		if err != nil {
			s.FailNow("Failed to change modified time for file", err)
		}
	}

	cclfmap = make(map[string][]*cclfFileMetadata)
	filePath = "../shared_files/cclf9/"
	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	assert.Nil(err)
	cclflist = cclfmap["A0001_18"]
	for _, cclf := range cclflist {
		// check for the new modification time
		assert.Equal(modtimeAfter, cclf.deliveryDate)
	}

	cclfmap = make(map[string][]*cclfFileMetadata)
	skipped = 0
	filePath = "../shared_files/cclf_All/"
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

func (s *CLITestSuite) TestImportCCLFDirectory() {
	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	db.Unscoped().Delete(&models.CCLFBeneficiary{})
	db.Unscoped().Delete(&models.CCLFFile{})

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	s.setPendingDeletionDir()

	args := []string{"bcda", "import-cclf-directory", "--directory", "../shared_files/cclf/"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 3 files.")
	assert.Contains(buf.String(), "Failed to import 0 files.")
	assert.Contains(buf.String(), "Skipped 0 files.")

	buf.Reset()
	db.Unscoped().Delete(&models.CCLFBeneficiary{})
	db.Unscoped().Delete(&models.CCLFFile{})
	s.resetFiles("../shared_files/cclf/")

	// dir has 4 files, but 2 will be ignored because of bad file names.
	args = []string{"bcda", "import-cclf-directory", "--directory", "../shared_files/cclf_BadFileNames/"}
	err = s.testApp.Run(args)
	assert.NotNil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 2 files.")
	assert.Contains(buf.String(), "Failed to import 1 files.")
	assert.Contains(buf.String(), "Skipped 2 files.")
	buf.Reset()

	s.resetFiles("../shared_files/cclf_BadFileNames/")
}
func (s *CLITestSuite) TestCleanupCCLF() {
	cclfmap := make(map[string][]*cclfFileMetadata)
	s.setPendingDeletionDir()

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
		filePath:     "../shared_files/cclf/T.A0001.ACO.ZC0Y18.D181120.T1000011",
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
		filePath:     "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009",
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
		filePath:  "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010",
		imported:  true,
	}
	cclfmap["A0001_18"] = []*cclfFileMetadata{cclf0metadata, cclf8metadata, cclf9metadata}
	cleanupCCLF(cclfmap)

	files, err := ioutil.ReadDir(os.Getenv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", os.Getenv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual(s.T(), "T.A0001.ACO.ZC0Y18.D181120.T1000011", file.Name())
	}
	s.resetFiles("../shared_files/cclf/")
}

func (s *CLITestSuite) TestImportCCLFDirectory_SplitFiles() {
	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	db.Unscoped().Delete(&models.CCLFBeneficiary{})
	db.Unscoped().Delete(&models.CCLFFile{})

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	args := []string{"bcda", "import-cclf-directory", "--directory", "../shared_files/cclf_split/"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 3 files.")
	assert.Contains(buf.String(), "Failed to import 0 files.")
	assert.Contains(buf.String(), "Skipped 0 files.")

	s.resetFiles("../shared_files/cclf_split/")
}

func (s *CLITestSuite) TestDeleteDirectory() {
	assert := assert.New(s.T())
	dirToDelete := "../shared_files/doomedDirectory"
	s.makeDirToDelete(dirToDelete)
	defer os.Remove(dirToDelete)

	f, err := os.Open(dirToDelete)
	assert.Nil(err)
	files, err := f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(4, len(files))

	filesDeleted, err := deleteDirectoryContents(dirToDelete)
	assert.Equal(4, filesDeleted)
	assert.Nil(err)

	f, err = os.Open(dirToDelete)
	assert.Nil(err)
	files, err = f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(0, len(files))

	filesDeleted, err = deleteDirectoryContents("This/Does/not/Exist")
	assert.Equal(0, filesDeleted)
	assert.NotNil(err)
}

func (s *CLITestSuite) TestDeleteDirectoryContents() {
	assert := assert.New(s.T())
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	dirToDelete := "../shared_files/doomedDirectory"
	s.makeDirToDelete(dirToDelete)
	defer os.Remove(dirToDelete)

	args := []string{"bcda", "delete-dir-contents", "--dirToDelete", dirToDelete}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), fmt.Sprintf("Successfully Deleted 4 files from %v", dirToDelete))
	buf.Reset()

	// File, not a directory
	args = []string{"bcda", "delete-dir-contents", "--dirToDelete", "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"}
	err = s.testApp.Run(args)
	assert.NotNil(err)
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

	os.Setenv("TESTDELETEDIRECTORY", "NOT/A/REAL/DIRECTORY")
	args = []string{"bcda", "delete-dir-contents", "--envvar", "TESTDELETEDIRECTORY"}
	err = s.testApp.Run(args)
	assert.NotNil(err)
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

}

func (s *CLITestSuite) makeDirToDelete(filePath string) {
	assert := assert.New(s.T())
	dirToDelete := filePath
	err := os.Mkdir(dirToDelete, os.ModePerm)
	assert.Nil(err)

	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe1.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe2.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe3.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe4.txt"))
	assert.Nil(err)
}

func (s *CLITestSuite) setPendingDeletionDir() {
	err := os.Setenv("PENDING_DELETION_DIR", "/go/src/github.com/CMSgov/bcda-app/bcda/pending_delete_dir")
	if err != nil {
		s.FailNow("failed to set the PENDING_DELETION_DIR env variable,", err)
	}
	cclfDeletion := os.Getenv("PENDING_DELETION_DIR")
	err = os.MkdirAll(cclfDeletion, 0744)
	if err != nil {
		s.FailNow("failed to create the pending deletion directory,", err)
	}
}

func (s *CLITestSuite) resetFiles(resetPath string) {
	err := filepath.Walk(os.Getenv("PENDING_DELETION_DIR"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				s.FailNow("error in walkfunc,", err)
			}

			if info.IsDir() {
				return nil
			}
			err = os.Rename(path, resetPath+info.Name())
			if err != nil {
				s.FailNow("error in moving files,", err)
			}
			return nil
		})
	if err != nil {
		s.FailNow("error in walkfunc,", err)
	}
}
