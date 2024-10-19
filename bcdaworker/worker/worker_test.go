package worker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
	log "github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
)

var logHook *test.Hook
var oldLogger logrus.FieldLogger
var GlobalLogger *logrus.Logger

type WorkerTestSuite struct {
	suite.Suite
	testACO *models.ACO

	// test params
	jobID      int
	stagingDir string
	payloadDir string
	tempDir    string
	cclfFile   *models.CCLFFile

	db *sql.DB
	r  repository.Repository
	w  Worker

	logctx context.Context
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

	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}

	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", fmt.Sprintf("%s/%s", tempDir, "PAYLOAD"))
	conf.SetEnv(s.T(), "FHIR_STAGING_DIR", fmt.Sprintf("%s/%s", tempDir, "STAGING"))
	conf.SetEnv(s.T(), "FHIR_TEMP_DIR", fmt.Sprintf("%s/%s", tempDir, "TEMP"))
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")

	// Set up the logger since we're using the real client
	client.SetLogger(log.BBWorker)
	oldLogger = log.Worker

	ctx := context.Background()
	s.logctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
}

func (s *WorkerTestSuite) SetupTest() {
	s.jobID = generateUniqueJobID(s.T(), s.db, s.testACO.UUID)
	s.cclfFile = &models.CCLFFile{CCLFNum: 8, ACOCMSID: *s.testACO.CMSID, Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	s.stagingDir = fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_STAGING_DIR"), s.jobID)
	s.payloadDir = fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), s.jobID)
	s.tempDir = fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_TEMP_DIR"), uuid.NewRandom())

	postgrestest.CreateCCLFFile(s.T(), s.db, s.cclfFile)
	os.RemoveAll(s.stagingDir)

	if err := os.MkdirAll(s.stagingDir, os.ModePerm); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.MkdirAll(s.payloadDir, os.ModePerm); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.MkdirAll(s.tempDir, os.ModePerm); err != nil {
		s.FailNow(err.Error())
	}

	// Due to test not being able to handle a FieldLogger and our log package not storing the logger,
	// we have to recreate the logger so we have access to both
	// logger := logrus.New()
	// log.Worker = logger.WithFields(logrus.Fields{
	// 	"unitTest": true,
	// })
	// logHook = test.NewLocal(logger)
}

func (s *WorkerTestSuite) TearDownTest() {
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, *s.testACO.CMSID)
	os.RemoveAll(s.stagingDir)
}

func (s *WorkerTestSuite) TearDownSuite() {
	postgrestest.DeleteACO(s.T(), s.db, s.testACO.UUID)
	os.RemoveAll(conf.GetEnv("FHIR_STAGING_DIR"))
	os.RemoveAll(conf.GetEnv("FHIR_PAYLOAD_DIR"))
	os.RemoveAll(conf.GetEnv("FHIR_TEMP_DIR"))

	// Reset worker logger to original logger
	log.Worker = oldLogger
}

func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}

func (s *WorkerTestSuite) TestWriteResourcesToFile() {
	tests := []struct {
		resource      string
		jobKeysCount  int
		fileCount     int
		expectedCount int
		err           error
	}{
		{"ExplanationOfBenefit", 1, 1, 33, nil},
		{"Coverage", 1, 1, 3, nil},
		{"Patient", 1, 1, 1, nil},
		{"Claim", 1, 1, 1, nil},
		{"ClaimResponse", 1, 1, 1, nil},
		{"UnsupportedResource", 1, 0, 0, errors.Errorf("unsupported resouce")},
	}

	for _, tt := range tests {
		ctx, jobArgs, bbc := SetupWriteResourceToFile(s, tt.resource)
		jobKeys, err := writeBBDataToFile(ctx, s.r, bbc, *s.testACO.CMSID, cryptoRandInt63(), jobArgs, s.tempDir)
		if tt.err == nil {
			assert.NoError(s.T(), err)
		} else {
			assert.Error(s.T(), err)
		}
		files, err := os.ReadDir(s.tempDir)
		assert.NoError(s.T(), err)
		assert.Len(s.T(), jobKeys, tt.jobKeysCount)
		assert.Len(s.T(), files, tt.fileCount)
		VerifyFileContent(s.T(), files, tt.resource, tt.expectedCount, s.tempDir)
	}
}

func SetupWriteResourceToFile(s *WorkerTestSuite, resource string) (context.Context, models.JobEnqueueArgs, *client.MockBlueButtonClient) {
	bbc := client.MockBlueButtonClient{}
	since, transactionTime := time.Now().Add(-24*time.Hour).Format(time.RFC3339Nano), time.Now()
	claimsWindow := client.ClaimsWindow{LowerBound: time.Now().Add(-365 * 24 * time.Hour), UpperBound: time.Now().Add(-180 * 24 * time.Hour)}
	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: resource, Since: since, TransactionTime: transactionTime, ClaimsWindow: claimsWindow}
	var cclfBeneficiaryIDs []string
	beneID := "a1000050699"
	bbc.MBI = &beneID
	cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneID, BlueButtonID: beneID}
	postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
	cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	jobArgs.BeneficiaryIDs = cclfBeneficiaryIDs
	ctx := context.Background()
	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)

	switch resource {
	case "ExplanationOfBenefit":
		bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetExplanationOfBenefit", jobArgs, beneID, claimsWindowMatcher(claimsWindow.LowerBound, claimsWindow.UpperBound)).Return(bbc.GetBundleData("ExplanationOfBenefit", beneID))
	case "Coverage":
		bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetCoverage", jobArgs, beneID).Return(bbc.GetBundleData("Coverage", beneID))
	case "Patient":
		bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetPatient", jobArgs, beneID).Return(bbc.GetBundleData("Patient", beneID))
	case "Claim":
		bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetClaim", jobArgs, beneID, claimsWindowMatcher(claimsWindow.LowerBound, claimsWindow.UpperBound)).Return(bbc.GetBundleData("Claim", beneID))
	case "ClaimResponse":
		bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneID))
		bbc.On("GetClaimResponse", jobArgs, beneID, claimsWindowMatcher(claimsWindow.LowerBound, claimsWindow.UpperBound)).Return(bbc.GetBundleData("ClaimResponse", beneID))

	}
	return ctx, jobArgs, &bbc
}

func VerifyFileContent(t *testing.T, files []fs.DirEntry, resource string, expectedCount int, tempDir string) {
	for _, f := range files {
		filePath := fmt.Sprintf("%s/%s", tempDir, f.Name())
		file, err := os.Open(filePath)
		if err != nil {
			t.FailNow()
		}
		defer func() {
			assert.NoError(t, file.Close())
			assert.NoError(t, os.Remove(filePath))
		}()

		scanner := bufio.NewScanner(file)

		for i := 0; i < expectedCount; i++ {
			assert.True(t, scanner.Scan())
			var jsonOBJ map[string]interface{}
			err := json.Unmarshal(scanner.Bytes(), &jsonOBJ)
			assert.Nil(t, err)
			assert.Equal(t, resource, jsonOBJ["resourceType"])
			if resource == "ExplanationOfBenefit" || resource == "Coverage" {
				assert.NotNil(t, jsonOBJ["status"], "JSON should contain a value for `status`.")
				assert.NotNil(t, jsonOBJ["type"], "JSON should contain a value for `type`.")
			}
		}
		scan := scanner.Scan()
		assert.False(t, scan, "There should be only %d entries in the file.", expectedCount)
	}
}

func (s *WorkerTestSuite) TestWriteEmptyResourceToFile() {
	transactionTime := time.Now()

	bbc := client.MockBlueButtonClient{}
	beneficiaryID := "abcdef12000"
	var cclfBeneficiaryIDs []string

	bbc.MBI = &beneficiaryID
	cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneficiaryID, BlueButtonID: beneficiaryID}
	postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
	cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))

	bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneficiaryID))

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: transactionTime, ACOID: s.testACO.UUID.String()}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefit", jobArgs, "abcdef12000", client.ClaimsWindow{}).Return(bbc.GetBundleData("ExplanationOfBenefitEmpty", "abcdef12000"))
	jobKeys, err := writeBBDataToFile(s.logctx, s.r, &bbc, *s.testACO.CMSID, cryptoRandInt63(), jobArgs, s.tempDir)
	assert.EqualValues(s.T(), "blank.ndjson", jobKeys[0].FileName)
	assert.NoError(s.T(), err)
}

func (s *WorkerTestSuite) TestWriteEOBDataToFileWithErrorsBelowFailureThreshold() {
	origFailPct := conf.GetEnv("EXPORT_FAIL_PCT")
	defer conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", origFailPct)
	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "70")
	transactionTime := time.Now()
	bbc := client.MockBlueButtonClient{}
	beneficiaryIDs := []string{"abcdef10000", "abcdef11000", "abcdef12000"}
	var cclfBeneficiaryIDs []string

	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		bbc.MBI = &beneficiaryID
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneficiaryID, BlueButtonID: beneficiaryID}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
		bbc.On("GetPatientByMbi", cclfBeneficiary.MBI).Return(bbc.GetData("Patient", beneficiaryID))
	}

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: transactionTime, ACOID: s.testACO.UUID.String()}
	// Set up the mock function to return the expected values
	bbc.On("GetExplanationOfBenefit", jobArgs, "abcdef10000", claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", jobArgs, "abcdef11000", claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", jobArgs, "abcdef12000", claimsWindowMatcher()).Return(bbc.GetBundleData("ExplanationOfBenefit", "abcdef12000"))
	jobKeys, err := writeBBDataToFile(s.logctx, s.r, &bbc, *s.testACO.CMSID, cryptoRandInt63(), jobArgs, s.tempDir)
	assert.NotEqual(s.T(), "blank.ndjson", jobKeys[0].FileName)
	assert.Contains(s.T(), jobKeys[1].FileName, "error.ndjson")
	assert.Len(s.T(), jobKeys, 2)
	assert.NoError(s.T(), err)
	errorFilePath := fmt.Sprintf("%s/%s", s.tempDir, jobKeys[1].FileName)
	fData, err := os.ReadFile(errorFilePath)
	assert.NoError(s.T(), err)

	ooResp := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","diagnostics":"Error retrieving ExplanationOfBenefit for beneficiary MBI abcdef10000 in ACO %s"}]}
	{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","diagnostics":"Error retrieving ExplanationOfBenefit for beneficiary MBI abcdef11000 in ACO %s"}]}`, s.testACO.UUID, s.testACO.UUID)

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

	var cclfBeneficiaryIDs []string
	beneficiaryIDs := []string{"a1000089833", "a1000065301", "a1000012463"}
	for i := 0; i < len(beneficiaryIDs); i++ {
		beneficiaryID := beneficiaryIDs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: beneficiaryID, BlueButtonID: beneficiaryID}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	}

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: transactionTime, ACOID: s.testACO.UUID.String()}
	bbc := client.MockBlueButtonClient{}
	// Set up the mock function to return the expected values

	bbc.On("GetExplanationOfBenefit", jobArgs, beneficiaryIDs[0], claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.On("GetExplanationOfBenefit", jobArgs, beneficiaryIDs[1], claimsWindowMatcher()).Return(nil, errors.New("error"))
	bbc.MBI = &beneficiaryIDs[0]
	bbc.On("GetPatientByMbi", beneficiaryIDs[0]).Return(bbc.GetData("Patient", beneficiaryIDs[0]))

	bbc.MBI = &beneficiaryIDs[1]
	bbc.On("GetPatientByMbi", beneficiaryIDs[1]).Return(bbc.GetData("Patient", beneficiaryIDs[1]))

	jobArgs.BeneficiaryIDs = cclfBeneficiaryIDs
	err := createDir(s.tempDir)
	assert.NoError(s.T(), err)
	jobKeys, err := writeBBDataToFile(s.logctx, s.r, &bbc, *s.testACO.CMSID, cryptoRandInt63(), jobArgs, s.tempDir)
	assert.Len(s.T(), jobKeys, 1)
	assert.Contains(s.T(), err.Error(), "Number of failed requests has exceeded threshold")

	files, err := os.ReadDir(s.tempDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(files))

	errorFilePath := fmt.Sprintf("%s/%s", s.tempDir, files[0].Name())
	fData, err := os.ReadFile(errorFilePath)
	assert.NoError(s.T(), err)
	ooResp := fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","diagnostics":"Error retrieving ExplanationOfBenefit for beneficiary MBI a1000089833 in ACO %s"}]}
	{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","diagnostics":"Error retrieving ExplanationOfBenefit for beneficiary MBI a1000065301 in ACO %s"}]}`, s.testACO.UUID, s.testACO.UUID)

	// Since our error file ends with a new line character, we need
	// to remove it in order so split OperationOutcome responses by newline character
	fData = fData[:len(fData)-1]
	assertEqualErrorFiles(s.T(), ooResp, string(fData))

	// Assert cmsID and jobID fields are being added to the logs
	bbc.AssertExpectations(s.T())
	// should not have requested third beneficiary EOB because failure threshold was reached after second
	bbc.AssertNotCalled(s.T(), "GetExplanationOfBenefit", jobArgs, beneficiaryIDs[2], claimsWindowMatcher())
}

func (s *WorkerTestSuite) TestWriteEOBDataToFile_BlueButtonIDNotFound() {
	origFailPct := conf.GetEnv("EXPORT_FAIL_PCT")
	defer conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", origFailPct)
	conf.SetEnv(s.T(), "EXPORT_FAIL_PCT", "51")

	bbc := client.MockBlueButtonClient{}
	bbc.On("GetPatientByMbi", mock.AnythingOfType("string")).Return("", errors.New("No beneficiary found for MBI"))

	badMBIs := []string{"ab000000001", "ab000000002"}
	var cclfBeneficiaryIDs []string
	for i := 0; i < len(badMBIs); i++ {
		mbi := badMBIs[i]
		cclfBeneficiary := models.CCLFBeneficiary{FileID: s.cclfFile.ID, MBI: mbi, BlueButtonID: ""}
		postgrestest.CreateCCLFBeneficiary(s.T(), s.db, &cclfBeneficiary)
		cclfBeneficiaryIDs = append(cclfBeneficiaryIDs, strconv.FormatUint(uint64(cclfBeneficiary.ID), 10))
	}

	jobArgs := models.JobEnqueueArgs{ID: s.jobID, ResourceType: "ExplanationOfBenefit", BeneficiaryIDs: cclfBeneficiaryIDs, TransactionTime: time.Now(), ACOID: s.testACO.UUID.String()}
	jobKeys, err := writeBBDataToFile(s.logctx, s.r, &bbc, *s.testACO.CMSID, cryptoRandInt63(), jobArgs, s.tempDir)
	assert.Len(s.T(), jobKeys, 1)
	assert.Equal(s.T(), jobKeys[0].FileName, "blank.ndjson")
	assert.Contains(s.T(), err.Error(), "Number of failed requests has exceeded threshold")

	files, err := os.ReadDir(s.tempDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(files))

	dataFilePath := fmt.Sprintf("%s/%s", s.tempDir, files[1].Name())
	d, err := os.ReadFile(dataFilePath)
	if err != nil {
		s.FailNow(err.Error())
	}
	// Should be empty
	s.Empty(d)

	errorFilePath := fmt.Sprintf("%s/%s", s.tempDir, files[0].Name())
	d, err = os.ReadFile(errorFilePath)
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
		assert.Equal(s.T(), "not-found", issue["code"])
		assert.Equal(s.T(), fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary MBI %s", cclfBeneficiary.MBI), issue["diagnostics"])
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
	appendErrorToFile(s.logctx, s.testACO.UUID.String(),
		fhircodes.IssueTypeCode_CODE_INVALID,
		"", "", s.tempDir)

	filePath := fmt.Sprintf("%s/%s-error.ndjson", s.tempDir, s.testACO.UUID)
	fData, err := os.ReadFile(filePath)
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
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusPending,
		JobCount:   1,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	complete, err := CheckJobCompleteAndCleanup(s.logctx, s.r, j.ID)
	assert.Nil(s.T(), err)
	assert.False(s.T(), complete)

	jobArgs := models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{"10000", "11000"},
		ResourceType:   "ExplanationOfBenefit",
		BBBasePath:     constants.TestFHIRPath,
		TransactionID:  uuid.New(),
	}

	ctx, _ := log.SetCtxLogger(s.logctx, "job_id", j.ID)
	ctx, logger := log.SetCtxLogger(ctx, "transaction_id", jobArgs.TransactionID)
	logHook = test.NewLocal(testUtils.GetLogger(logger))

	err = s.w.ProcessJob(ctx, cryptoRandInt63(), j, jobArgs)

	entries := logHook.AllEntries()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), entries[0].Data, "cms_id")
	assert.Contains(s.T(), entries[0].Data, "job_id")
	assert.Contains(s.T(), entries[0].Data, "transaction_id")

	_, err = CheckJobCompleteAndCleanup(ctx, s.r, j.ID)
	assert.Nil(s.T(), err)
	completedJob, err := s.r.GetJobByID(context.Background(), j.ID)
	fmt.Printf("%+v", completedJob)
	assert.Nil(s.T(), err)
	// As this test actually connects to BB, we can't be sure it will succeed
	assert.Contains(s.T(), []models.JobStatus{models.JobStatusFailed, models.JobStatusCompleted}, completedJob.Status)
}

func (s *WorkerTestSuite) TestProcessJobUpdateJobCheckStatus() {
	ctx := context.Background()
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusPending,
		JobCount:   1,
	}

	jobArgs := models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{"10000", "11000"},
		ResourceType:   "ExplanationOfBenefit",
		BBBasePath:     constants.TestFHIRPath,
	}
	r := &repository.MockRepository{}
	defer r.AssertExpectations(s.T())
	r.On("GetACOByUUID", testUtils.CtxMatcher, j.ACOID).Return(s.testACO, nil)
	r.On("UpdateJobStatusCheckStatus", testUtils.CtxMatcher, uint(jobArgs.ID), models.JobStatusPending, models.JobStatusInProgress).Return(errors.New("failure"))
	w := &worker{r}
	err := w.ProcessJob(ctx, cryptoRandInt63(), j, jobArgs)
	assert.NotNil(s.T(), err)

}

func (s *WorkerTestSuite) TestProcessJobACOUUID() {
	ctx := context.Background()
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusPending,
		JobCount:   1,
	}

	jobArgs := models.JobEnqueueArgs{
		ID:             int(j.ID),
		ACOID:          j.ACOID.String(),
		BeneficiaryIDs: []string{"10000", "11000"},
		ResourceType:   "ExplanationOfBenefit",
		BBBasePath:     constants.TestFHIRPath,
	}

	r := &repository.MockRepository{}
	defer r.AssertExpectations(s.T())
	r.On("GetACOByUUID", testUtils.CtxMatcher, j.ACOID).Return(nil, repository.ErrJobNotFound)
	w := &worker{r}
	err := w.ProcessJob(ctx, cryptoRandInt63(), j, jobArgs)
	assert.NotNil(s.T(), err)

}

func (s *WorkerTestSuite) TestCreateDir() {
	err := createDir("/proc/invalid_path") //non-existant dir
	assert.Error(s.T(), err)
	err = createDir("2") //fine
	assert.NoError(s.T(), err)
}

func (s *WorkerTestSuite) TestCompressFilesGzipLevel() {
	//In short, none of these should produce errors when being run.
	tempDir1, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}
	tempDir2, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}

	os.Setenv("COMPRESSION_LEVEL", "potato")
	err = compressFiles(s.logctx, tempDir1, tempDir2)
	assert.NoError(s.T(), err)

	os.Setenv("COMPRESSION_LEVEL", "1")
	err = compressFiles(s.logctx, tempDir1, tempDir2)
	assert.NoError(s.T(), err)

	os.Setenv("COMPRESSION_LEVEL", "11")
	err = compressFiles(s.logctx, tempDir1, tempDir2)
	assert.NoError(s.T(), err)

}

func (s *WorkerTestSuite) TestCompressFiles() {
	//negative cases.
	err := compressFiles(s.logctx, "/", "fake_dir")
	assert.Error(s.T(), err)
	err = compressFiles(s.logctx, "/proc/fakedir", "fake_dir")
	assert.Error(s.T(), err)

	//positive case, create two temporary directories + a file, and move a file between them.
	tempDir1, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}
	tempDir2, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}
	_, err = os.CreateTemp(tempDir1, "")
	if err != nil {
		s.FailNow(err.Error())
	}
	err = compressFiles(s.logctx, tempDir1, tempDir2)
	assert.NoError(s.T(), err)
	files, _ := os.ReadDir(tempDir2)
	assert.Len(s.T(), files, 1)
	files, _ = os.ReadDir(tempDir1)
	assert.Len(s.T(), files, 1)

	//One more negative case, when the destination is not able to be moved.
	err = compressFiles(s.logctx, tempDir2, "/proc/fakedir")
	assert.Error(s.T(), err)

}

func (s *WorkerTestSuite) TestProcessJob_NoBBClient() {
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
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
		BBBasePath:     constants.TestFHIRPath,
	}

	origBBCert := conf.GetEnv("BB_CLIENT_CERT_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origBBCert)
	conf.UnsetEnv(s.T(), "BB_CLIENT_CERT_FILE")

	assert.Contains(s.T(), s.w.ProcessJob(s.logctx, cryptoRandInt63(), j, jobArgs).Error(), "could not create Blue Button client")
}

func (s *WorkerTestSuite) TestJobCancelledTerminalStatus() {
	ctx := context.Background()
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
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
		BBBasePath:     constants.TestFHIRPath,
	}

	processJobErr := s.w.ProcessJob(ctx, cryptoRandInt63(), j, jobArgs)
	completedJob, _ := s.r.GetJobByID(ctx, j.ID)

	// cancelled parent job status should not update after failed queuejob
	assert.Contains(s.T(), processJobErr.Error(), "job was not updated, no match found")
	assert.Equal(s.T(), models.JobStatusCancelled, completedJob.Status)
}

func (s *WorkerTestSuite) TestProcessJobInvalidDirectory() {

	tests := []struct {
		name        string
		stagingFail bool
		payloadFail bool
		tempDirFail bool
	}{
		{"TempDirFailure", false, false, true},
		{"StagingDirFailure", true, false, false},
		{"PayloadDirFailure", false, true, false},
		{"NoFailure", false, false, false},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			// Use multiple defers to ensure that the conf.GetEnv gets evaluated prior to us
			// modifying the value.
			defer conf.SetEnv(s.T(), "FHIR_STAGING_DIR", conf.GetEnv("FHIR_STAGING_DIR"))
			defer conf.SetEnv(s.T(), "FHIR_PAYL0AD_DIR", conf.GetEnv("FHIR_PAYL0AD_DIR"))
			defer conf.SetEnv(s.T(), "FHIR_TEMP_DIR", conf.GetEnv("FHIR_TEMP_DIR"))
			staging, err := os.MkdirTemp("", "*")
			assert.NoError(s.T(), err)
			payload, err := os.MkdirTemp("", "*")
			assert.NoError(s.T(), err)
			tmp, err := os.MkdirTemp("", "*")
			assert.NoError(s.T(), err)
			if tt.stagingFail {
				staging = "/proc/invalid_path"
			}
			if tt.payloadFail {
				payload = "/proc/invalid_path"
			}
			if tt.tempDirFail {
				tmp = "/proc/invalid_path"
			}

			conf.SetEnv(s.T(), "FHIR_STAGING_DIR", staging)
			conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", payload)
			conf.SetEnv(s.T(), "FHIR_TEMP_DIR", tmp)
			ctx := context.Background()
			j := models.Job{
				ACOID:      uuid.Parse(constants.TestACOID),
				RequestURL: "/api/v1/Patient/$export",
				Status:     models.JobStatusInProgress,
				JobCount:   1,
			}
			postgrestest.CreateJobs(s.T(), s.db, &j)

			jobArgs := models.JobEnqueueArgs{
				ID:             int(j.ID),
				ACOID:          j.ACOID.String(),
				BeneficiaryIDs: []string{"10000", "11000"},
				ResourceType:   "ExplanationOfBenefit",
				BBBasePath:     constants.TestFHIRPath,
			}

			processJobErr := s.w.ProcessJob(ctx, cryptoRandInt63(), j, jobArgs)

			// cancelled parent job status should not update after failed queuejob
			if tt.payloadFail || tt.stagingFail || tt.tempDirFail {
				assert.Contains(s.T(), processJobErr.Error(), "could not create")
			} else {
				assert.NoError(s.T(), processJobErr)
			}

		})
	}

}

func (s *WorkerTestSuite) TestCheckJobCompleteAndCleanup() {
	// Use multiple defers to ensure that the conf.GetEnv gets evaluated prior to us
	// modifying the value.
	defer conf.SetEnv(s.T(), "FHIR_STAGING_DIR", conf.GetEnv("FHIR_STAGING_DIR"))
	defer conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", conf.GetEnv("FHIR_PAYLOAD_DIR"))
	staging, err := os.MkdirTemp("", "*")
	assert.NoError(s.T(), err)
	payload, err := os.MkdirTemp("", "*")
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
			randInt64, err := cryptoRandInt64()
			if err != nil {
				s.FailNow("Failed to generate random int64")
			}
			jobID, _ := safecast.ToUint(randInt64)

			sDir := fmt.Sprintf("%s/%d", staging, jobID)
			pDir := fmt.Sprintf("%s/%d", payload, jobID)

			assert.NoError(t, os.Mkdir(sDir, os.ModePerm))
			assert.NoError(t, os.Mkdir(pDir, os.ModePerm))

			f, err := os.CreateTemp(sDir, "")
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

			completed, err := CheckJobCompleteAndCleanup(s.logctx, repository, jobID)
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

	noBasePath := models.JobEnqueueArgs{ID: int(cryptoRandInt31())}
	jobNotFound := models.JobEnqueueArgs{ID: int(cryptoRandInt31()), BBBasePath: uuid.New()}
	dbErr := models.JobEnqueueArgs{ID: int(cryptoRandInt31()), BBBasePath: uuid.New()}
	jobCancelled := models.JobEnqueueArgs{ID: int(cryptoRandInt31()), BBBasePath: uuid.New()}
	jobFailed := models.JobEnqueueArgs{ID: int(cryptoRandInt31()), BBBasePath: uuid.New()}
	validJob := models.JobEnqueueArgs{ID: int(cryptoRandInt31()), BBBasePath: uuid.New()}
	r.On("GetJobByID", testUtils.CtxMatcher, uint(jobNotFound.ID)).Return(nil, repository.ErrJobNotFound)
	r.On("GetJobByID", testUtils.CtxMatcher, uint(dbErr.ID)).Return(nil, fmt.Errorf("some db error"))
	r.On("GetJobByID", testUtils.CtxMatcher, uint(jobCancelled.ID)).
		Return(&models.Job{ID: uint(jobCancelled.ID), Status: models.JobStatusCancelled}, nil)
	r.On("GetJobByID", testUtils.CtxMatcher, uint(jobFailed.ID)).
		Return(&models.Job{ID: uint(jobCancelled.ID), Status: models.JobStatusFailed}, nil)

	r.On("GetJobByID", testUtils.CtxMatcher, uint(validJob.ID)).
		Return(&models.Job{ID: uint(validJob.ID), Status: models.JobStatusPending}, nil)
	r.On("GetJobKey", testUtils.CtxMatcher, uint(validJob.ID), int64(0)).
		Return(nil, repository.ErrJobKeyNotFound)

	// Return existing job key, indicating que job was already processed.
	r.On("GetJobKey", testUtils.CtxMatcher, uint(validJob.ID), int64(1)).
		Return(&models.JobKey{ID: uint(validJob.ID)}, nil)

	r.On("GetJobKey", testUtils.CtxMatcher, uint(validJob.ID), int64(2)).
		Return(nil, fmt.Errorf("some db error"))

	defer func() {
		r.AssertExpectations(s.T())
		// Shouldn't be called because we already determined the job is invalid
		r.AssertNotCalled(s.T(), "GetJobByID", testUtils.CtxMatcher, uint(noBasePath.ID))
	}()

	j, err := w.ValidateJob(ctx, 0, noBasePath)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrNoBasePathSet.Error())

	j, err = w.ValidateJob(ctx, 0, jobNotFound)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrParentJobNotFound.Error())

	j, err = w.ValidateJob(ctx, 0, dbErr)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), "some db error")

	j, err = w.ValidateJob(ctx, 0, jobCancelled)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrParentJobCancelled.Error())

	j, err = w.ValidateJob(ctx, 0, jobFailed)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrParentJobFailed.Error())

	j, err = w.ValidateJob(ctx, 0, validJob)
	assert.NoError(s.T(), err)
	assert.EqualValues(s.T(), validJob.ID, j.ID)

	j, err = w.ValidateJob(ctx, 1, validJob)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), ErrQueJobProcessed.Error())

	j, err = w.ValidateJob(ctx, 2, validJob)
	assert.Nil(s.T(), j)
	assert.Contains(s.T(), err.Error(), "could not retrieve job key from database: some db error")
}

func (s *WorkerTestSuite) TestCreateJobKeys() {
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusPending,
		JobCount:   1,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	complete, err := CheckJobCompleteAndCleanup(s.logctx, s.r, j.ID)
	assert.Nil(s.T(), err)
	assert.False(s.T(), complete)

	keys := []models.JobKey{
		{JobID: 1, FileName: models.BlankFileName, ResourceType: "Patient"},
		{JobID: 1, FileName: uuid.New() + ".ndjson", ResourceType: "Coverage"},
	}
	err = createJobKeys(s.logctx, s.r, keys, j.ID)
	assert.NoError(s.T(), err)
	for i := 0; i < len(keys); i++ {
		job, _ := postgrestest.GetJobKey(s.db, int(keys[i].JobID))
		assert.NotEmpty(s.T(), job)
	}
}

func (s *WorkerTestSuite) TestCreateJobKeys_CreateJobKeysError() {
	r := &repository.MockRepository{}

	keys := []models.JobKey{
		{JobID: 1, FileName: models.BlankFileName, ResourceType: "Patient"},
		{JobID: 1, FileName: uuid.New() + ".ndjson", ResourceType: "Coverage"},
	}

	r.On("CreateJobKeys", testUtils.CtxMatcher, mock.Anything).Return(fmt.Errorf("some db error"))
	err := createJobKeys(s.logctx, r, keys, 1234)
	assert.ErrorContains(s.T(), err, "Error creating job key records for filenames")
	assert.ErrorContains(s.T(), err, keys[0].FileName)
	assert.ErrorContains(s.T(), err, keys[1].FileName)
	assert.ErrorContains(s.T(), err, "some db error")
}

func (s *WorkerTestSuite) TestCreateJobKeys_JobCompleteError() {
	r := &repository.MockRepository{}

	keys := []models.JobKey{
		{JobID: 1, FileName: models.BlankFileName, ResourceType: "Patient"},
		{JobID: 1, FileName: uuid.New() + ".ndjson", ResourceType: "Coverage"},
	}
	r.On("CreateJobKeys", testUtils.CtxMatcher, mock.Anything).Return(nil)
	r.On("GetJobByID", testUtils.CtxMatcher, uint(1)).Return(nil, fmt.Errorf("some db error"))
	err := createJobKeys(s.logctx, r, keys, 1)
	assert.ErrorContains(s.T(), err, "Failed retrieve job by id (Job 1)")
	assert.ErrorContains(s.T(), err, "some db error")
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

// cryptoRandInt64 generates a cryptographically secure random int64.
func cryptoRandInt64() (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func cryptoRandInt31() int32 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<31))
	if err != nil {
		panic(err) // handle error appropriately
	}
	return int32(n.Int64())
}
