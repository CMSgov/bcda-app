package worker

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
)

type WorkerTestSuite struct {
	suite.Suite
	testACO *models.ACO

	// test params
	jobID      int
	stagingDir string
	cclfFile   *models.CCLFFile

	db *sql.DB
	r  repository.Repository
	w  Worker
}

func (s *WorkerTestSuite) SetupSuite() {
	s.db = database.Connection
	s.r = postgres.NewRepository(s.db)
	s.w = NewWorker(s.db)

	cmsID := "A1B2C" // Some unique ID that should be unique to this test

	s.testACO = &models.ACO{
		UUID:  uuid.NewUUID(),
		CMSID: &cmsID,
		Name:  "ACO_FOR_WORKER_TEST",
	}

	postgrestest.CreateACO(s.T(), s.db, *s.testACO)

	tempDir, err := ioutil.TempDir("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}

	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", tempDir)
	conf.SetEnv(s.T(), "FHIR_STAGING_DIR", tempDir)
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")
	conf.SetEnv(s.T(), "ATO_PUBLIC_KEY_FILE", "../../shared_files/ATO_public.pem")
	conf.SetEnv(s.T(), "ATO_PRIVATE_KEY_FILE", "../../shared_files/ATO_private.pem")
}

func (s *WorkerTestSuite) SetupTest() {
	s.jobID = generateUniqueJobID(s.T(), s.db, s.testACO.UUID)
	s.cclfFile = &models.CCLFFile{CCLFNum: 8, ACOCMSID: *s.testACO.CMSID, Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	s.stagingDir = fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID)

	postgrestest.CreateCCLFFile(s.T(), s.db, s.cclfFile)
	os.RemoveAll(s.stagingDir)

	if err := os.MkdirAll(s.stagingDir, os.ModePerm); err != nil {
		s.FailNow(err.Error())
	}
}

func (s *WorkerTestSuite) TearDownTest() {
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, *s.testACO.CMSID)
	os.RemoveAll(s.stagingDir)
}

func (s *WorkerTestSuite) TearDownSuite() {
	postgrestest.DeleteACO(s.T(), s.db, s.testACO.UUID)
	os.RemoveAll(conf.GetEnv("FHIR_STAGING_DIR"))
	os.RemoveAll(conf.GetEnv("FHIR_PAYLOAD_DIR"))
}

func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}

func (s *WorkerTestSuite) TestWriteResourceToFile() {
	bbc := client.MockBlueButtonClient{}
	since, transactionTime := time.Now().Add(-24*time.Hour).Format(time.RFC3339Nano), time.Now()
	claimsWindow := client.ClaimsWindow{LowerBound: time.Now().Add(-365 * 24 * time.Hour), UpperBound: time.Now().Add(-180 * 24 * time.Hour)}

	var cclfBeneficiaryIDs []string
	for _, beneID := range []string{"a1000003701", "a1000050699"} {
		bbc.MBI = &beneID
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneID, BlueButtonID: beneID}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetExplanationOfBenefit", beneID, strconv.Itoa(s.jobID), *s.testACO.CMSID, since, transactionTime,
			claimsWindowMatcher(claimsWindow.LowerBound, claimsWindow.UpperBound)).Return(bbc.GetBundleData("ExplanationOfBenefit", beneID))
		bbc.On("GetCoverage", beneID, strconv.Itoa(s.jobID), *s.testACO.CMSID, since, transactionTime).Return(bbc.GetBundleData("Coverage", beneID))
		bbc.On("GetPatient", beneID, strconv.Itoa(s.jobID), *s.testACO.CMSID, since, transactionTime).Return(bbc.GetBundleData("Patient", beneID))
	}

	tests := []struct {
		resource       string
		expectedCount  int
		expectZeroSize bool
	}{
		{"ExplanationOfBenefit", 66, false},
		{"Coverage", 6, false},
		{"Patient", 2, false},
		{"SomeUnsupportedResource", 0, true},
	}

	for _, tt := range tests {
		s.T().Run(tt.resource, func(t *testing.T) {
			jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: tt.resource, BeneficiaryIDs: cclfBeneficiaryIDs,
				Since: since, TransactionTime: transactionTime, ClaimsWindow: claimsWindow}
			uuid, size, err := writeBBDataToFile(context.Background(), s.r, &bbc, *s.testACO.CMSID, jobArgs)
			if tt.expectZeroSize {
				assert.EqualValues(t, 0, size)
			} else {
				assert.NotEqual(t, int64(0), size)
			}

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
				filePath := fmt.Sprintf("%s/%d/%s", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID, f.Name())
				file, err := os.Open(filePath)
				if err != nil {
					s.FailNow(err.Error())
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

func (s *WorkerTestSuite) TestWriteEmptyResourceToFile() {
	transactionTime := time.Now()

	bbc := client.MockBlueButtonClient{}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefit", "abcdef12000", strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher()).Return(bbc.GetBundleData("ExplanationOfBenefitEmpty", "abcdef12000"))
	beneficiaryID := "abcdef12000"
	var cclfBeneficiaryIDs []string

	bbc.MBI = &beneficiaryID
	cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneficiaryID, BlueButtonID: beneficiaryID}
	postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
	cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", beneficiaryID))

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: transactionTime, ACOID: s.testACO.UUID.String()}
	_, size, err := writeBBDataToFile(context.Background(), s.r, &bbc, *s.testACO.CMSID, jobArgs)
	assert.EqualValues(s.T(), 0, size)
	assert.NoError(s.T(), err)
}

func (s *WorkerTestSuite) TestWriteEOBDataToFileWithErrorsBelowFailureThreshold() {
	origFailPct := conf.GetEnv("EXPORT_FAIL_PCT")
	defer conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", origFailPct)
	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "70")
	transactionTime := time.Now()

	bbc := client.MockBlueButtonClient{}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefit", "abcdef10000", strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", "abcdef11000", strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", "abcdef12000", strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher()).Return(bbc.GetBundleData("ExplanationOfBenefit", "abcdef12000"))
	beneficiaryIDs := []string{"abcdef10000", "abcdef11000", "abcdef12000"}
	var cclfBeneficiaryIDs []string

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		bbc.MBI = &beneficiaryID
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneficiaryID, BlueButtonID: beneficiaryID}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", beneficiaryID))
	}

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: transactionTime, ACOID: s.testACO.UUID.String()}
	fileUUID, size, err := writeBBDataToFile(context.Background(), s.r, &bbc, *s.testACO.CMSID, jobArgs)
	assert.NotEqual(s.T(), int64(0), size)
	assert.NoError(s.T(), err)

	errorFilePath := fmt.Sprintf("%s/%d/%s-error.ndjson", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID, fileUUID)
	fData, err := ioutil.ReadFile(errorFilePath)
	assert.NoError(s.T(), err)

	ooResp := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary MBI abcdef10000 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary MBI abcdef10000 in ACO %s"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary MBI abcdef11000 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary MBI abcdef11000 in ACO %s"}}]}`, s.testACO.UUID, s.testACO.UUID, s.testACO.UUID, s.testACO.UUID)

	// Since our error file ends with a new line character, we need
	// to remove it in order so split OperationOutcome responses by newline character
	fData = fData[:len(fData)-1]
	assertEqualErrorFiles(s.T(), ooResp, string(fData))

	bbc.AssertExpectations(s.T())
}

func (s *WorkerTestSuite) TestWriteEOBDataToFileWithErrorsAboveFailureThreshold() {
	origFailPct := conf.GetEnv("EXPORT_FAIL_PCT")
	defer conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", origFailPct)
	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "60")
	transactionTime := time.Now()

	bbc := client.MockBlueButtonClient{}
	// Set up the mock function to return the expected values
	beneficiaryIDs := []string{"a1000089833", "a1000065301", "a1000012463"}
	bbc.On("GetExplanationOfBenefit", beneficiaryIDs[0], strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", beneficiaryIDs[1], strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.MBI = &beneficiaryIDs[0]
	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(beneficiaryIDs[0])).Return(bbc.GetData("Patient", beneficiaryIDs[0]))
	bbc.MBI = &beneficiaryIDs[1]
	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(beneficiaryIDs[1])).Return(bbc.GetData("Patient", beneficiaryIDs[1]))
	var cclfBeneficiaryIDs []string

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneficiaryID, BlueButtonID: beneficiaryID}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	}

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: transactionTime, ACOID: s.testACO.UUID.String()}
	_, _, err := writeBBDataToFile(context.Background(), s.r, &bbc, *s.testACO.CMSID, jobArgs)
	assert.Equal(s.T(), "number of failed requests has exceeded threshold", err.Error())

	files, err := ioutil.ReadDir(s.stagingDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(files))

	errorFilePath := fmt.Sprintf("%s/%d/%s", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID, files[0].Name())
	fData, err := ioutil.ReadFile(errorFilePath)
	assert.NoError(s.T(), err)

	ooResp := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary MBI a1000089833 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary MBI a1000089833 in ACO %s"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","details":{"coding":[{"system":"http://hl7.org/fhir/ValueSet/operation-outcome","code":"Blue Button Error","display":"Error retrieving ExplanationOfBenefit for beneficiary MBI a1000065301 in ACO %s"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary MBI a1000065301 in ACO %s"}}]}`, s.testACO.UUID, s.testACO.UUID, s.testACO.UUID, s.testACO.UUID)

	// Since our error file ends with a new line character, we need
	// to remove it in order so split OperationOutcome responses by newline character
	fData = fData[:len(fData)-1]
	assertEqualErrorFiles(s.T(), ooResp, string(fData))

	bbc.AssertExpectations(s.T())
	// should not have requested third beneficiary EOB because failure threshold was reached after second
	bbc.AssertNotCalled(s.T(), "GetExplanationOfBenefit", beneficiaryIDs[2], strconv.Itoa(s.jobID), *s.testACO.CMSID, "", transactionTime, claimsWindowMatcher())
}

func (s *WorkerTestSuite) TestWriteEOBDataToFile_BlueButtonIDNotFound() {
	origFailPct := conf.GetEnv("EXPORT_FAIL_PCT")
	defer conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", origFailPct)
	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "51")

	bbc := client.MockBlueButtonClient{}
	bbc.On("GetPatientByIdentifierHash", mock.AnythingOfType("string")).Return("", errors.New("No beneficiary found for MBI"))

	badMBIs := []string{"ab000000001", "ab000000002"}
	var cclfBeneficiaryIDs []string
	for i := 0; i < len(badMBIs); i++ {
		mbi := badMBIs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: mbi, BlueButtonID: ""}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	}

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: time.Now(), ACOID: s.testACO.UUID.String()}
	_, _, err := writeBBDataToFile(context.Background(), s.r, &bbc, *s.testACO.CMSID, jobArgs)
	assert.EqualError(s.T(), err, "number of failed requests has exceeded threshold")

	files, err := ioutil.ReadDir(s.stagingDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(files))

	dataFilePath := fmt.Sprintf("%s/%d/%s", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID, files[1].Name())
	d, err := ioutil.ReadFile(dataFilePath)
	if err != nil {
		s.FailNow(err.Error())
	}
	// Should be empty
	s.Empty(d)

	errorFilePath := fmt.Sprintf("%s/%d/%s", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID, files[0].Name())
	d, err = ioutil.ReadFile(errorFilePath)
	if err != nil {
		s.FailNow(err.Error())
	}

	errorFileScanner := bufio.NewScanner(bytes.NewReader(d))
	for _, cclfBeneID := range cclfBeneficiaryIDs {
		beneID, err := strconv.ParseUint(cclfBeneID, 10, 64)
		assert.NoError(s.T(), err)
		cclfBeneficiary, err := s.r.GetCCLFBeneficiaryByID(context.Background(), uint(beneID))
		assert.NoError(s.T(), err)
		assert.True(s.T(), errorFileScanner.Scan())
		var jsonObj map[string]interface{}
		err = json.Unmarshal(errorFileScanner.Bytes(), &jsonObj)
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), "OperationOutcome", jsonObj["resourceType"])
		issues := jsonObj["issue"].([]interface{})
		issue := issues[0].(map[string]interface{})
		assert.Equal(s.T(), "error", issue["severity"])
		details := issue["details"].(map[string]interface{})
		assert.Equal(s.T(), fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary MBI %s", cclfBeneficiary.MBI), details["text"])
	}
	assert.False(s.T(), errorFileScanner.Scan(), "There should be only 2 entries in the file.")

	bbc.AssertExpectations(s.T())
}

func (s *WorkerTestSuite) TestGetFailureThreshold() {
	origFailPct := conf.GetEnv("EXPORT_FAIL_PCT")
	defer conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", origFailPct)

	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "60")
	assert.Equal(s.T(), 60.0, getFailureThreshold())

	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "-1")
	assert.Equal(s.T(), 0.0, getFailureThreshold())

	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "500")
	assert.Equal(s.T(), 100.0, getFailureThreshold())

	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "zero")
	assert.Equal(s.T(), 50.0, getFailureThreshold())
}

func (s *WorkerTestSuite) TestAppendErrorToFile() {
	appendErrorToFile(context.Background(), s.testACO.UUID.String(),
		fhircodes.IssueTypeCode_CODE_INVALID,
		"", "", s.jobID)

	filePath := fmt.Sprintf("%s/%d/%s-error.ndjson", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID, s.testACO.UUID)
	fData, err := ioutil.ReadFile(filePath)
	assert.NoError(s.T(), err)

	type oo struct {
		ResourceType string `json:"resourceType"`
		Issues       []struct {
			Severity string `json:"severity"`
		} `json:"issue"`
	}
	var obj oo
	assert.NoError(s.T(), json.Unmarshal(fData, &obj))
	assert.Equal(s.T(), "OperationOutcome", obj.ResourceType)
	assert.Equal(s.T(), "error", obj.Issues[0].Severity)

	os.Remove(filePath)
}

func (s *WorkerTestSuite) TestProcessJobEOB() {
	ctx := context.Background()
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusPending,
		JobCount:   1,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	complete, err := checkJobCompleteAndCleanup(ctx, s.r, j.ID)
	assert.Nil(s.T(), err)
	assert.False(s.T(), complete)

	jobArgs := models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{"10000", "11000"},
		ResourceType:   "ExplanationOfBenefit",
		BBBasePath:     "/v1/fhir",
	}

	err = s.w.ProcessJob(ctx, j, jobArgs)
	assert.Nil(s.T(), err)
	_, err = checkJobCompleteAndCleanup(ctx, s.r, j.ID)
	assert.Nil(s.T(), err)
	completedJob, err := s.r.GetJobByID(context.Background(), j.ID)
	fmt.Printf("%+v", completedJob)
	assert.Nil(s.T(), err)
	// As this test actually connects to BB, we can't be sure it will succeed
	assert.Contains(s.T(), []models.JobStatus{models.JobStatusFailed, models.JobStatusCompleted}, completedJob.Status)
	assert.Equal(s.T(), 1, completedJob.CompletedJobCount)
}

func (s *WorkerTestSuite) TestProcessJob_NoBBClient() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.db, j.ID)

	jobArgs := models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{},
		ResourceType:   "Patient",
		BBBasePath:     "/v1/fhir",
	}

	origBBCert := conf.GetEnv("BB_CLIENT_CERT_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origBBCert)
	conf.UnsetEnv(s.T(), "BB_CLIENT_CERT_FILE")

	assert.Contains(s.T(), s.w.ProcessJob(context.Background(), j, jobArgs).Error(), "could not create Blue Button client")
}

func (s *WorkerTestSuite) TestJobCancelledTerminalStatus() {
	ctx := context.Background()
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     models.JobStatusCancelled,
		JobCount:   1,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	jobArgs := models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{"10000", "11000"},
		ResourceType:   "ExplanationOfBenefit",
		BBBasePath:     "/v1/fhir",
	}

	processJobErr := s.w.ProcessJob(ctx, j, jobArgs)
	completedJob, _ := s.r.GetJobByID(context.Background(), j.ID)

	// cancelled parent job status should not update after failed queuejob
	assert.Contains(s.T(), processJobErr.Error(), "job was not updated, no match found")
	assert.Equal(s.T(), models.JobStatusCancelled, completedJob.Status)
}

func (s *WorkerTestSuite) TestCheckJobCompleteAndCleanup() {
	// Use multiple defers to ensure that the conf.GetEnv gets evaluated prior to us
	// modifying the value.
	defer conf.SetEnv(s.T(), "FHIR_STAGING_DIR", conf.GetEnv("FHIR_STAGING_DIR"))
	defer conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", conf.GetEnv("FHIR_PAYLOAD_DIR"))

	staging, err := ioutil.TempDir("", "*")
	assert.NoError(s.T(), err)
	payload, err := ioutil.TempDir("", "*")
	assert.NoError(s.T(), err)
	conf.SetEnv(s.T(), "FHIR_STAGING_DIR", staging)
	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", payload)

	tests := []struct {
		name      string
		status    models.JobStatus
		jobCount  int
		jobKeys   int
		completed bool
	}{
		{"PendingButComplete", models.JobStatusPending, 1, 1, true},
		{"PendingNotComplete", models.JobStatusPending, 10, 1, false},
		{"AlreadyCompleted", models.JobStatusCompleted, 1, 1, true},
		{"Cancelled", models.JobStatusCancelled, 1, 1, true},
		{"Failed", models.JobStatusFailed, 1, 1, true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			jobID := uint(rand.Uint64())

			sDir := fmt.Sprintf("%s/%d", staging, jobID)
			pDir := fmt.Sprintf("%s/%d", payload, jobID)

			assert.NoError(t, os.Mkdir(sDir, os.ModePerm))
			assert.NoError(t, os.Mkdir(pDir, os.ModePerm))

			f, err := ioutil.TempFile(sDir, "")
			assert.NoError(t, err)
			assert.NoError(t, f.Close())

			j := &models.Job{ID: jobID, Status: tt.status, JobCount: tt.jobCount}
			repository := &repository.MockRepository{}
			defer repository.AssertExpectations(t)
			repository.On("GetJobByID", testUtils.CtxMatcher, jobID).Return(j, nil)

			// A job previously marked as a terminal status (Completed, Cancelled, or Failed) will bypass all of these calls
			if !isTerminalStatus(tt.status) {
				repository.On("GetJobKeyCount", testUtils.CtxMatcher, jobID).Return(tt.jobKeys, nil)
				if tt.completed {
					repository.On("UpdateJobStatus", testUtils.CtxMatcher, j.ID, models.JobStatusCompleted).
						Return(nil)
				}
			}

			completed, err := checkJobCompleteAndCleanup(context.Background(), repository, jobID)
			assert.NoError(t, err)
			assert.Equal(t, tt.completed, completed)

			// Terminal Status job should've bypassed all of these calls. Therefore any data will remain.
			if tt.completed && !isTerminalStatus(tt.status) {
				_, err := os.Stat(sDir)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func isTerminalStatus(status models.JobStatus) bool {
	switch status {
	case models.JobStatusCompleted,
		models.JobStatusCancelled,
		models.JobStatusFailed:
		return true
	}
	return false
}

func (s *WorkerTestSuite) TestValidateJob() {
	ctx := context.Background()
	r := &repository.MockRepository{}
	w := &worker{r}

	noBasePath := models.JobEnqueueArgs{ID: int(rand.Int31())}
	jobNotFound := models.JobEnqueueArgs{ID: int(rand.Int31()), BBBasePath: uuid.New()}
	dbErr := models.JobEnqueueArgs{ID: int(rand.Int31()), BBBasePath: uuid.New()}
	jobCancelled := models.JobEnqueueArgs{ID: int(rand.Int31()), BBBasePath: uuid.New()}
	validJob := models.JobEnqueueArgs{ID: int(rand.Int31()), BBBasePath: uuid.New()}
	r.On("GetJobByID", testUtils.CtxMatcher, uint(jobNotFound.ID)).Return(nil, repository.ErrJobNotFound)
	r.On("GetJobByID", testUtils.CtxMatcher, uint(dbErr.ID)).Return(nil, fmt.Errorf("some db error"))
	r.On("GetJobByID", testUtils.CtxMatcher, uint(jobCancelled.ID)).
		Return(&models.Job{ID: uint(jobCancelled.ID), Status: models.JobStatusCancelled}, nil)
	r.On("GetJobByID", testUtils.CtxMatcher, uint(validJob.ID)).
		Return(&models.Job{ID: uint(validJob.ID), Status: models.JobStatusPending}, nil)

	defer func() {
		r.AssertExpectations(s.T())
		// Shouldn't be called because we already determined the job is invalid
		r.AssertNotCalled(s.T(), "GetJobByID", testUtils.CtxMatcher, uint(noBasePath.ID))
	}()

	j, err := w.ValidateJob(ctx, noBasePath)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrNoBasePathSet.Error())

	j, err = w.ValidateJob(ctx, jobNotFound)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrParentJobNotFound.Error())

	j, err = w.ValidateJob(ctx, dbErr)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), "some db error")

	j, err = w.ValidateJob(ctx, jobCancelled)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrParentJobCancelled.Error())

	j, err = w.ValidateJob(ctx, validJob)
	assert.NoError(s.T(), err)
	assert.EqualValues(s.T(), validJob.ID, j.ID)
}

func generateUniqueJobID(t *testing.T, db *sql.DB, acoID uuid.UUID) int {
	j := models.Job{
		ACOID:      acoID,
		RequestURL: "/some/request/URL",
	}
	postgrestest.CreateJobs(t, db, &j)
	return int(j.ID)
}

// first argument is lowerBound, second argument is upperBound
func claimsWindowMatcher(times ...time.Time) (matcher interface{}) {
	expected := client.ClaimsWindow{}
	switch len(times) {
	case 2:
		expected.UpperBound = times[1]
		fallthrough
	case 1:
		expected.LowerBound = times[0]
	}

	return mock.MatchedBy(func(actual client.ClaimsWindow) bool {
		return expected.LowerBound.Equal(actual.LowerBound) &&
			expected.UpperBound.Equal(actual.UpperBound)
	})
}

func assertEqualErrorFiles(t *testing.T, expected, actual string) {
	// Since we have multiple OperationOutcome responses to handle
	// we need to split them and deserialize them individually.
	// By placing them in a map[string]interface{} we can ensure equality
	// even if the order of the JSON fields are different.
	var expectedOO, actualOO []map[string]interface{}
	for _, part := range strings.Split(expected, "\n") {
		var obj map[string]interface{}
		fmt.Println(part)
		assert.NoError(t, json.Unmarshal([]byte(part), &obj))
		expectedOO = append(expectedOO, obj)
	}

	for _, part := range strings.Split(actual, "\n") {
		var obj map[string]interface{}
		assert.NoError(t, json.Unmarshal([]byte(part), &obj))
		actualOO = append(actualOO, obj)
	}

	assert.Equal(t, expectedOO, actualOO)
}
