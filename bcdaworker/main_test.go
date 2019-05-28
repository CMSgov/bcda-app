package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type MainTestSuite struct {
	suite.Suite
	reset func()
}

func (s *MainTestSuite) SetupSuite() {
	s.reset = testUtils.SetUnitTestKeysForAuth()
}

func (s *MainTestSuite) TearDownSuite() {
	s.reset()
}

func (s *MainTestSuite) SetupTest() {
	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	os.Setenv("FHIR_STAGING_DIR", "data/test")
	os.Setenv("BB_CLIENT_CERT_FILE", "../shared_files/bb-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../shared_files/bb-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../shared_files/localhost.crt")
	os.Setenv("ATO_PUBLIC_KEY_FILE", "../shared_files/ATO_public.pem")
	os.Setenv("ATO_PRIVATE_KEY_FILE", "../shared_files/ATO_private.pem")
	models.InitializeGormModels()
}

func (s *MainTestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func TestWriteEOBDataToFile(t *testing.T) {
	db := database.GetGORMDbConnection()
	defer db.Close()
	bbc := testUtils.BlueButtonClient{}
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262b07"
	jobID := "1"
	staging := fmt.Sprintf("%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID)
	cclfFile := models.CCLFFile{CCLFNum: 8, ACOCMSID: "12345", Timestamp: time.Now(), PerformanceYear: 19, Name: "T.A12345.ACO.ZC8Y19.D191120.T1012309"}
	db.Create(&cclfFile)
	defer db.Delete(&cclfFile)

	// clean out the data dir before beginning this test
	os.RemoveAll(staging)
	testUtils.CreateStaging(jobID)
	beneficiaryIDs := []string{"1000003701", "1000050699"}
	var cclfBeneficiaryIDs []string
	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: cclfFile.ID, HICN: beneficiaryID, MBI: "whatever", BlueButtonID: beneficiaryID}
		db.Create(&cclfBeneficiary)
		defer db.Delete(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		bbc.On("GetExplanationOfBenefitData", beneficiaryIDs[i]).Return(bbc.GetData("ExplanationOfBenefit", beneficiaryID))
	}

	_, err := writeBBDataToFile(&bbc, acoID, cclfBeneficiaryIDs, jobID, "ExplanationOfBenefit")
	if err != nil {
		t.Fail()
	}

	files, err := ioutil.ReadDir(staging)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))

	for _, f := range files {
		fmt.Println(f.Name())
		filePath := fmt.Sprintf("%s/%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID, f.Name())
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)

		// 33 entries in test EOB data returned by bbc.getData, times two beneficiaries
		for i := 0; i < 66; i++ {
			assert.True(t, scanner.Scan())
			var jsonOBJ map[string]interface{}
			err := json.Unmarshal(scanner.Bytes(), &jsonOBJ)
			assert.Nil(t, err)
			assert.NotNil(t, jsonOBJ["fullUrl"], "JSON should contain a value for `fullUrl`.")
			assert.NotNil(t, jsonOBJ["resource"], "JSON should contain a value for `resource`.")
		}
		assert.False(t, scanner.Scan(), "There should be only 66 entries in the file.")

		bbc.AssertExpectations(t)

		file.Close()
		os.Remove(filePath)
	}
}

func TestWriteEOBDataToFileNoClient(t *testing.T) {
	_, err := writeBBDataToFile(nil, "9c05c1f8-349d-400f-9b69-7963f2262b08", []string{"20000", "21000"}, "1", "ExplanationOfBenefit")
	assert.NotNil(t, err)
}

func TestWriteEOBDataToFileInvalidACO(t *testing.T) {
	bbc := testUtils.BlueButtonClient{}
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262zzz"
	beneficiaryIDs := []string{"10000", "11000"}

	_, err := writeBBDataToFile(&bbc, acoID, beneficiaryIDs, "1", "ExplanationOfBenefit")
	assert.NotNil(t, err)
}

func TestWriteEOBDataToFileWithErrorsBelowFailureThreshold(t *testing.T) {

	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)
	os.Setenv("EXPORT_FAIL_PCT", "70")

	bbc := testUtils.BlueButtonClient{}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefitData", "10000").Return("", errors.New("error"))
	bbc.On("GetExplanationOfBenefitData", "11000").Return("", errors.New("error"))
	bbc.On("GetExplanationOfBenefitData", "12000").Return(bbc.GetData("ExplanationOfBenefit", "12000"))
	acoID := "387c3a62-96fa-4d93-a5d0-fd8725509dd9"
	beneficiaryIDs := []string{"10000", "11000", "12000"}
	var cclfBeneficiaryIDs []string

	db := database.GetGORMDbConnection()
	defer db.Close()
	cclfFile := models.CCLFFile{CCLFNum: 8, ACOCMSID: "12345", Timestamp: time.Now(), PerformanceYear: 19, Name: "T.A12345.ACO.ZC8Y19.D191120.T1012309"}
	db.Create(&cclfFile)
	defer db.Delete(&cclfFile)

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: cclfFile.ID, HICN: beneficiaryID, MBI: "whatever", BlueButtonID: beneficiaryID}
		db.Create(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		defer db.Delete(&cclfBeneficiary)
	}
	jobID := "1"
	testUtils.CreateStaging(jobID)

	fileName, err := writeBBDataToFile(&bbc, acoID, cclfBeneficiaryIDs, jobID, "ExplanationOfBenefit")
	if err != nil {
		t.Fail()
	}

	filePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, acoID)
	fData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fail()
	}

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 10000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 10000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 11000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 11000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}}]}`
	assert.Equal(t, ooResp+"\n", string(fData))
	bbc.AssertExpectations(t)

	os.Remove(fmt.Sprintf("%s/%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID, fileName))
	os.Remove(filePath)
}

func TestWriteEOBDataToFileWithErrorsAboveFailureThreshold(t *testing.T) {
	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)
	os.Setenv("EXPORT_FAIL_PCT", "60")

	bbc := testUtils.BlueButtonClient{}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefitData", "1000089833").Return("", errors.New("error"))
	bbc.On("GetExplanationOfBenefitData", "1000065301").Return("", errors.New("error"))
	acoID := "387c3a62-96fa-4d93-a5d0-fd8725509dd9"
	beneficiaryIDs := []string{"1000089833", "1000065301", "1000012463"}
	var cclfBeneficiaryIDs []string
	db := database.GetGORMDbConnection()
	defer db.Close()
	cclfFile := models.CCLFFile{CCLFNum: 8, ACOCMSID: "12345", Timestamp: time.Now(), PerformanceYear: 19, Name: "T.A12345.ACO.ZC8Y19.D191120.T1012309"}
	db.Create(&cclfFile)
	defer db.Delete(&cclfFile)

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: cclfFile.ID, HICN: beneficiaryID, MBI: "whatever", BlueButtonID: beneficiaryID}
		db.Create(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		defer db.Delete(&cclfBeneficiary)
	}

	jobID := "1"
	testUtils.CreateStaging(jobID)

	_, err := writeBBDataToFile(&bbc, acoID, cclfBeneficiaryIDs, jobID, "ExplanationOfBenefit")
	assert.Equal(t, "number of failed requests has exceeded threshold", err.Error())

	filePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, acoID)
	fData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fail()
	}

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 1000089833 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 1000089833 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 1000065301 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 1000065301 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}}]}`
	assert.Equal(t, ooResp+"\n", string(fData))
	bbc.AssertExpectations(t)
	// should not have requested third beneficiary EOB because failure threshold was reached after second
	bbc.AssertNotCalled(t, "GetExplanationOfBenefitData", "1000012463")

	os.Remove(fmt.Sprintf("%s/%s/%s.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, acoID))
	os.Remove(filePath)
}

func TestGetFailureThreshold(t *testing.T) {
	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)

	os.Setenv("EXPORT_FAIL_PCT", "60")
	assert.Equal(t, 60.0, getFailureThreshold())

	os.Setenv("EXPORT_FAIL_PCT", "-1")
	assert.Equal(t, 0.0, getFailureThreshold())

	os.Setenv("EXPORT_FAIL_PCT", "500")
	assert.Equal(t, 100.0, getFailureThreshold())

	os.Setenv("EXPORT_FAIL_PCT", "zero")
	assert.Equal(t, 50.0, getFailureThreshold())
}

func TestAppendErrorToFile(t *testing.T) {

	acoID := "328e83c3-bc46-4827-836c-0ba0c713dc7d"
	jobID := "1"
	testUtils.CreateStaging(jobID)
	appendErrorToFile(acoID, "", "", "", jobID)

	filePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, acoID)
	fData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fail()
	}

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"Error"}]}`

	assert.Equal(t, ooResp+"\n", string(fData))

	os.Remove(filePath)
}

func (s *MainTestSuite) TestProcessJobEOB() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := models.Job{
		ACOID:      uuid.Parse(constants.DEVACOUUID),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	db.Save(&j)

	complete, err := j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), complete)

	jobArgs := jobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		UserID:         j.UserID.String(),
		BeneficiaryIDs: []string{"10000", "11000"},
		ResourceType:   "ExplanationOfBenefit",
	}
	args, _ := json.Marshal(jobArgs)

	job := &que.Job{
		Type: "ProcessJob",
		Args: args,
	}
	fmt.Println("About to queue up the job")
	err = processJob(job)
	assert.Nil(s.T(), err)
	_, err = j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	var completedJob models.Job
	err = db.First(&completedJob, "ID = ?", jobArgs.ID).Error
	assert.Nil(s.T(), err)
	// As this test actually connects to BB, we can't be sure it will succeed
	assert.Contains(s.T(), []string{"Failed", "Completed"}, completedJob.Status)
}

func (s *MainTestSuite) TestSetupQueue() {
	setupQueue()
	os.Setenv("WORKER_POOL_SIZE", "7")
	setupQueue()
}
