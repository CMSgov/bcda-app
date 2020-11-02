package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/mock"

	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type MainTestSuite struct {
	suite.Suite
	reset   func()
	db      *gorm.DB
	testACO *models.ACO

	// test params
	jobID      int
	stagingDir string
	cclfFile   *models.CCLFFile
}

func (s *MainTestSuite) SetupSuite() {
	s.reset = testUtils.SetUnitTestKeysForAuth()
	s.db = database.GetGORMDbConnection()

	cmsID := "A1B2C" // Some unique ID that should be unique to this test

	s.testACO = &models.ACO{
		UUID:  uuid.NewUUID(),
		CMSID: &cmsID,
		Name:  "ACO_FOR_WORKER_TEST",
	}

	if err := s.db.Save(&s.testACO).Error; err != nil {
		s.FailNowf("Failed to add new ACO %s", err.Error())
	}

	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	os.Setenv("FHIR_STAGING_DIR", "data/test")
	os.Setenv("BB_CLIENT_CERT_FILE", "../shared_files/decrypted/bfd-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../shared_files/decrypted/bfd-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../shared_files/localhost.crt")
	os.Setenv("ATO_PUBLIC_KEY_FILE", "../shared_files/ATO_public.pem")
	os.Setenv("ATO_PRIVATE_KEY_FILE", "../shared_files/ATO_private.pem")
}

func (s *MainTestSuite) SetupTest() {
	s.jobID = generateUniqueJobID(s.T(), s.db, s.testACO.UUID)
	s.cclfFile = &models.CCLFFile{CCLFNum: 8, ACOCMSID: *s.testACO.CMSID, Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	s.stagingDir = fmt.Sprintf("%s/%d", os.Getenv("FHIR_STAGING_DIR"), s.jobID)

	s.NoError(s.db.Create(s.cclfFile).Error)
	os.RemoveAll(s.stagingDir)
	testUtils.CreateStaging(s.jobID)
}

func (s *MainTestSuite) TearDownTest() {
	s.db.Unscoped().Delete(&models.CCLFBeneficiary{}, "file_id = ?", s.cclfFile.ID)
	s.db.Unscoped().Delete(s.cclfFile)
	os.RemoveAll(s.stagingDir)
}

func (s *MainTestSuite) TearDownSuite() {
	s.reset()
	s.db.Unscoped().Where("aco_id = ?", s.testACO.UUID).Delete(&models.Job{})
	s.db.Unscoped().Delete(s.testACO)
	s.db.Close()
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func (s *MainTestSuite) TestWriteResourceToFile() {
	db := database.GetGORMDbConnection()
	defer db.Close()
	bbc := testUtils.BlueButtonClient{}
	since, transactionTime, serviceDate := time.Now().Add(-24*time.Hour).Format(time.RFC3339Nano), time.Now(), time.Now().Add(-180*24*time.Hour)

	var cclfBeneficiaryIDs []string
	for _, beneID := range []string{"a1000003701", "a1000050699"} {
		bbc.MBI = &beneID
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, HICN: "whatever", MBI: beneID, BlueButtonID: beneID}
		db.Create(&cclfBeneficiary)
		defer db.Delete(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetExplanationOfBenefit", beneID, strconv.Itoa(s.jobID), *s.testACO.CMSID, since, transactionTime, serviceDate).Return(bbc.GetBundleData("ExplanationOfBenefit", beneID))
		bbc.On("GetCoverage", beneID, strconv.Itoa(s.jobID), *s.testACO.CMSID, since, transactionTime).Return(bbc.GetBundleData("Coverage", beneID))
		bbc.On("GetPatient", beneID, strconv.Itoa(s.jobID), *s.testACO.CMSID, since, transactionTime).Return(bbc.GetBundleData("Patient", beneID))
	}

	tests := []struct {
		resource      string
		expectedCount int
	}{
		{"ExplanationOfBenefit", 66},
		{"Coverage", 6},
		{"Patient", 2},
		{"SomeUnsupportedResource", 0},
	}

	for _, tt := range tests {
		s.T().Run(tt.resource, func(t *testing.T) {
			jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: tt.resource, BeneficiaryIDs: cclfBeneficiaryIDs,
				Since: since, TransactionTime: transactionTime, ServiceDate: serviceDate}
			uuid, err := writeBBDataToFile(context.Background(), &bbc, db, *s.testACO.CMSID, jobArgs)

			files, err1 := ioutil.ReadDir(s.stagingDir)
			assert.NoError(t, err1)

			// If we don't expect any files, we must've encountered some error
			if tt.expectedCount == 0 {
				assert.Error(t, err)
				assert.Empty(t, uuid)
				files, err := ioutil.ReadDir(s.stagingDir)
				assert.NoError(t, err)
				assert.Len(t, files, 0)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, uuid)
			assert.Len(t, files, 1)

			for _, f := range files {
				filePath := fmt.Sprintf("%s/%d/%s", os.Getenv("FHIR_STAGING_DIR"), s.jobID, f.Name())
				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal(err)
				}
				defer func() {
					assert.NoError(t, file.Close())
					assert.NoError(t, os.Remove(filePath))
				}()

				scanner := bufio.NewScanner(file)

				for i := 0; i < tt.expectedCount; i++ {
					assert.True(t, scanner.Scan())
					var jsonOBJ map[string]interface{}
					err := json.Unmarshal(scanner.Bytes(), &jsonOBJ)
					assert.Nil(t, err)
					assert.Equal(t, tt.resource, jsonOBJ["resourceType"])
					if tt.resource == "ExplanationOfBenefit" || tt.resource == "Coverage" {
						assert.NotNil(t, jsonOBJ["status"], "JSON should contain a value for `status`.")
						assert.NotNil(t, jsonOBJ["type"], "JSON should contain a value for `type`.")
					}
				}
				assert.False(t, scanner.Scan(), "There should be only %d entries in the file.", tt.expectedCount)
			}
		})
	}

	// After running all of our subtests, we expect that our mocks were called as expected.
	bbc.AssertExpectations(s.T())
}

func (s *MainTestSuite) TestWriteEOBDataToFileWithErrorsBelowFailureThreshold() {
	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)
	os.Setenv("EXPORT_FAIL_PCT", "70")

	bbc := testUtils.BlueButtonClient{}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefit", "abcdef10000").Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", "abcdef11000").Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", "abcdef12000").Return(bbc.GetBundleData("ExplanationOfBenefit", "abcdef12000"))
	acoID, cmsID := s.testACO.UUID, *s.testACO.CMSID
	beneficiaryIDs := []string{"abcdef10000", "abcdef11000", "abcdef12000"}
	var cclfBeneficiaryIDs []string

	db := database.GetGORMDbConnection()
	defer db.Close()
	cclfFile := models.CCLFFile{CCLFNum: 8, ACOCMSID: cmsID, Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	db.Create(&cclfFile)
	defer db.Delete(&cclfFile)

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		bbc.MBI = &beneficiaryID
		cclfBeneficiary := models.CCLFBeneficiary{FileID: cclfFile.ID, HICN: "whatever", MBI: beneficiaryID, BlueButtonID: beneficiaryID}
		db.Create(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", beneficiaryID))
		defer db.Delete(&cclfBeneficiary)

	}
	jobID := generateUniqueJobID(s.T(), db, acoID)
	stagingDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_STAGING_DIR"), jobID)
	os.RemoveAll(stagingDir)
	testUtils.CreateStaging(jobID)

	jobArgs := models.JobEnqueueArgs{ID: jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: time.Now(), ACOID: acoID.String()}
	fileUUID, err := writeBBDataToFile(context.Background(), &bbc, db, cmsID, jobArgs)
	assert.NoError(s.T(), err)

	errorFilePath := fmt.Sprintf("%s/%d/%s-error.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, fileUUID)
	fData, err := ioutil.ReadFile(errorFilePath)
	assert.NoError(s.T(), err)

	ooResp := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary abcdef10000 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary abcdef10000 in ACO %s"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary abcdef11000 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary abcdef11000 in ACO %s"}}]}`, acoID, acoID, acoID, acoID)
	assert.Equal(s.T(), ooResp+"\n", string(fData))
	bbc.AssertExpectations(s.T())

	os.Remove(fmt.Sprintf("%s/%d/%s.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, fileUUID))
	os.Remove(errorFilePath)
}

func (s *MainTestSuite) TestWriteEOBDataToFileWithErrorsAboveFailureThreshold() {
	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)
	os.Setenv("EXPORT_FAIL_PCT", "60")

	bbc := testUtils.BlueButtonClient{}
	// Set up the mock function to return the expected values
	beneficiaryIDs := []string{"a1000089833", "a1000065301", "a1000012463"}
	bbc.On("GetExplanationOfBenefit", beneficiaryIDs[0]).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", beneficiaryIDs[1]).Return(nil, errors.New("error"))
	bbc.MBI = &beneficiaryIDs[0]
	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(beneficiaryIDs[0])).Return(bbc.GetData("Patient", beneficiaryIDs[0]))
	bbc.MBI = &beneficiaryIDs[1]
	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(beneficiaryIDs[1])).Return(bbc.GetData("Patient", beneficiaryIDs[1]))
	acoID, cmsID := s.testACO.UUID, *s.testACO.CMSID
	var cclfBeneficiaryIDs []string
	db := database.GetGORMDbConnection()
	defer db.Close()
	cclfFile := models.CCLFFile{CCLFNum: 8, ACOCMSID: "12345", Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	db.Create(&cclfFile)
	defer db.Delete(&cclfFile)

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: cclfFile.ID, HICN: "whatever", MBI: beneficiaryID, BlueButtonID: beneficiaryID}
		db.Create(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		defer db.Delete(&cclfBeneficiary)
	}

	jobID := generateUniqueJobID(s.T(), db, acoID)
	testUtils.CreateStaging(jobID)

	jobArgs := models.JobEnqueueArgs{ID: jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: time.Now(), ACOID: acoID.String()}
	_, err := writeBBDataToFile(context.Background(), &bbc, db, cmsID, jobArgs)
	assert.Equal(s.T(), "number of failed requests has exceeded threshold", err.Error())

	stagingDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_STAGING_DIR"), jobID)
	files, err := ioutil.ReadDir(stagingDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(files))

	errorFilePath := fmt.Sprintf("%s/%d/%s", os.Getenv("FHIR_STAGING_DIR"), jobID, files[0].Name())
	fData, err := ioutil.ReadFile(errorFilePath)
	assert.NoError(s.T(), err)

	ooResp := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary a1000089833 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary a1000089833 in ACO %s"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary a1000065301 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary a1000065301 in ACO %s"}}]}`, acoID, acoID, acoID, acoID)
	assert.Equal(s.T(), ooResp+"\n", string(fData))
	bbc.AssertExpectations(s.T())
	// should not have requested third beneficiary EOB because failure threshold was reached after second
	bbc.AssertNotCalled(s.T(), "GetExplanationOfBenefit", beneficiaryIDs[2])

	os.Remove(fmt.Sprintf("%s/%d/%s.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, acoID))
	os.Remove(errorFilePath)
}

func (s *MainTestSuite) TestWriteEOBDataToFile_BlueButtonIDNotFound() {
	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)
	os.Setenv("EXPORT_FAIL_PCT", "51")

	db := database.GetGORMDbConnection()
	defer db.Close()
	acoID, cmsID := s.testACO.UUID, *s.testACO.CMSID
	jobID := generateUniqueJobID(s.T(), db, acoID)
	stagingDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_STAGING_DIR"), jobID)
	cclfFile := models.CCLFFile{CCLFNum: 8, ACOCMSID: cmsID, Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	db.Create(&cclfFile)
	defer db.Delete(&cclfFile)

	bbc := testUtils.BlueButtonClient{}
	bbc.On("GetPatientByIdentifierHash", mock.AnythingOfType("string")).Return("", errors.New("No beneficiary found for MBI"))

	// clean out the data dir before beginning this test
	os.RemoveAll(stagingDir)
	testUtils.CreateStaging(jobID)
	badMBIs := []string{"ab000000001", "ab000000002"}
	var cclfBeneficiaryIDs []string
	for i := 0; i < len(badMBIs); i++ {
		mbi := badMBIs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: cclfFile.ID, HICN: "", MBI: mbi, BlueButtonID: ""}
		db.Create(&cclfBeneficiary)
		defer db.Delete(&cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	}

	jobArgs := models.JobEnqueueArgs{ID: jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: time.Now(), ACOID: acoID.String()}
	_, err := writeBBDataToFile(context.Background(), &bbc, db, cmsID, jobArgs)
	assert.EqualError(s.T(), err, "number of failed requests has exceeded threshold")

	files, err := ioutil.ReadDir(stagingDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(files))

	dataFilePath := fmt.Sprintf("%s/%d/%s", os.Getenv("FHIR_STAGING_DIR"), jobID, files[1].Name())
	dataFile, err := os.Open(dataFilePath)
	if err != nil {
		log.Fatal(err)
	}
	dataFileScanner := bufio.NewScanner(dataFile)
	// Should be empty
	assert.False(s.T(), dataFileScanner.Scan())
	dataFile.Close()

	errorFilePath := fmt.Sprintf("%s/%d/%s", os.Getenv("FHIR_STAGING_DIR"), jobID, files[0].Name())
	errorFile, err := os.Open(errorFilePath)
	if err != nil {
		log.Fatal(err)
	}
	errorFileScanner := bufio.NewScanner(errorFile)
	for _, cclfBeneID := range cclfBeneficiaryIDs {
		assert.True(s.T(), errorFileScanner.Scan())
		var jsonObj map[string]interface{}
		err := json.Unmarshal(errorFileScanner.Bytes(), &jsonObj)
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), "OperationOutcome", jsonObj["resourceType"])
		issues := jsonObj["issue"].([]interface{})
		issue := issues[0].(map[string]interface{})
		assert.Equal(s.T(), "error", issue["severity"])
		details := issue["details"].(map[string]interface{})
		assert.Equal(s.T(), fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary %s", cclfBeneID), details["text"])
	}
	assert.False(s.T(), errorFileScanner.Scan(), "There should be only 2 entries in the file.")
	errorFile.Close()

	bbc.AssertExpectations(s.T())

	os.Remove(dataFilePath)
	os.Remove(errorFilePath)
}

func (s *MainTestSuite) TestGetFailureThreshold() {
	origFailPct := os.Getenv("EXPORT_FAIL_PCT")
	defer os.Setenv("EXPORT_FAIL_PCT", origFailPct)

	os.Setenv("EXPORT_FAIL_PCT", "60")
	assert.Equal(s.T(), 60.0, getFailureThreshold())

	os.Setenv("EXPORT_FAIL_PCT", "-1")
	assert.Equal(s.T(), 0.0, getFailureThreshold())

	os.Setenv("EXPORT_FAIL_PCT", "500")
	assert.Equal(s.T(), 100.0, getFailureThreshold())

	os.Setenv("EXPORT_FAIL_PCT", "zero")
	assert.Equal(s.T(), 50.0, getFailureThreshold())
}

func (s *MainTestSuite) TestAppendErrorToFile() {
	db := database.GetGORMDbConnection()
	defer db.Close()

	acoID := s.testACO.UUID
	jobID := generateUniqueJobID(s.T(), db, acoID)
	testUtils.CreateStaging(jobID)
	appendErrorToFile(context.Background(), acoID.String(), "", "", "", jobID)

	filePath := fmt.Sprintf("%s/%d/%s-error.ndjson", os.Getenv("FHIR_STAGING_DIR"), jobID, acoID)
	fData, err := ioutil.ReadFile(filePath)
	assert.NoError(s.T(), err)

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"error"}]}`

	assert.Equal(s.T(), ooResp+"\n", string(fData))

	os.Remove(filePath)
}

func (s *MainTestSuite) TestProcessJobEOB() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// Verifies that we can handle an empty and specified basePath
	// TODO (BCDA-3895) - we should confirm that the job fails when we supply an empty basePath
	for _, basePath := range []string{"", "/v1/fhir"} {
		j := models.Job{
			ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
			RequestURL: "/api/v1/ExplanationOfBenefit/$export",
			Status:     "Pending",
			JobCount:   1,
		}
		db.Save(&j)

		complete, err := j.CheckCompletedAndCleanup(db)
		assert.Nil(s.T(), err)
		assert.False(s.T(), complete)

		jobArgs := models.JobEnqueueArgs{
			ID:             int(j.ID),
			ACOID:          j.ACOID.String(),
			BeneficiaryIDs: []string{"10000", "11000"},
			ResourceType:   "ExplanationOfBenefit",
			BBBasePath:     basePath,
		}
		args, _ := json.Marshal(jobArgs)

		job := &que.Job{
			Type: "ProcessJob",
			Args: args,
		}
		fmt.Println("About to queue up the job")
		err = processJob(job)
		assert.Nil(s.T(), err)
		_, err = j.CheckCompletedAndCleanup(db)
		assert.Nil(s.T(), err)
		var completedJob models.Job
		err = db.First(&completedJob, "ID = ?", jobArgs.ID).Error
		assert.Nil(s.T(), err)
		// As this test actually connects to BB, we can't be sure it will succeed
		assert.Contains(s.T(), []string{"Failed", "Completed"}, completedJob.Status)
	}
}

func (s *MainTestSuite) TestProcessJob_InvalidArgs() {
	j := que.Job{Args: []byte("{ this is not valid JSON }")}
	assert.EqualError(s.T(), processJob(&j), "invalid character 't' looking for beginning of object key string")
}

func (s *MainTestSuite) TestProcessJob_InvalidJobID() {
	qjArgs, _ := json.Marshal(models.JobEnqueueArgs{
		ID:             99999999,
		ACOID:          "00000000-0000-0000-0000-000000000000",
		BeneficiaryIDs: []string{},
		ResourceType:   "Patient",
	})

	qj := que.Job{
		Type: "ProcessJob",
		Args: qjArgs,
	}

	assert.Contains(s.T(), processJob(&qj).Error(), "could not retrieve job from database")
}

func (s *MainTestSuite) TestProcessJob_NoBBClient() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	db.Save(&j)

	qjArgs, _ := json.Marshal(models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{},
		ResourceType:   "Patient",
	})

	qj := que.Job{
		Type: "ProcessJob",
		Args: qjArgs,
	}

	origBBCert := os.Getenv("BB_CLIENT_CERT_FILE")
	defer os.Setenv("BB_CLIENT_CERT_FILE", origBBCert)
	os.Unsetenv("BB_CLIENT_CERT_FILE")

	assert.Contains(s.T(), processJob(&qj).Error(), "could not create Blue Button client")

	db.Unscoped().Delete(&j)
}

func (s *MainTestSuite) TestSetupQueue() {
	setupQueue()
	os.Setenv("WORKER_POOL_SIZE", "7")
	setupQueue()
}

func (s *MainTestSuite) TestUpdateJobStats() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := models.Job{
		ACOID:             uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL:        "",
		Status:            "",
		JobCount:          4,
		CompletedJobCount: 1,
	}
	db.Create(&j)
	updateJobStats(j.ID, db)
	db.First(&j, j.ID)
	assert.Equal(s.T(), 2, j.CompletedJobCount)
}

func (s *MainTestSuite) TestQueueJobWithNoParent() {
	retryCount := 10
	os.Setenv("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", strconv.Itoa(retryCount))
	tests := []struct {
		name        string
		errorCount  int32
		expectedErr error
	}{
		{"RetriesRemaining", int32(retryCount) - 1, errors.New("could not retrieve job from database: record not found")},
		{"RetriesExhausted", int32(retryCount), nil},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			qjArgs, _ := json.Marshal(models.JobEnqueueArgs{
				ID:             99999999, // JobID is not found in the db
				ACOID:          "00000000-0000-0000-0000-000000000000",
				BeneficiaryIDs: []string{},
				ResourceType:   "Patient",
			})

			qj := &que.Job{
				Type:       "ProcessJob",
				Args:       qjArgs,
				Priority:   1,
				ErrorCount: tt.errorCount,
			}

			err := processJob(qj)
			if tt.expectedErr != nil {
				assert.Equal(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func generateUniqueJobID(t *testing.T, db *gorm.DB, acoID uuid.UUID) int {
	j := models.Job{
		ACOID:      acoID,
		RequestURL: "/some/request/URL",
	}
	assert.NoError(t, db.Save(&j).Error)
	return int(j.ID)
}
