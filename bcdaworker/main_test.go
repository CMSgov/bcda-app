package main

import (
	"errors"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/url"
	"log"
    "strings"
	"os"
	"testing"
)

type MockBlueButtonClient struct {
	mock.Mock
}

type MockBlueButtonClientWithError struct {
	mock.Mock
}

type MainTestSuite struct {
	testUtils.AuthTestSuite
}

func (s *MainTestSuite) SetupTest() {
	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	os.Setenv("BB_CLIENT_CERT_FILE", "../shared_files/bb-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../shared_files/bb-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../shared_files/test-server-cert.pem")
	models.InitializeGormModels()
}

func (s *MainTestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func TestWriteEOBDataToFile(t *testing.T) {
	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	bbc := MockBlueButtonClient{}
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262b07"
	beneficiaryIDs := []string{"10000", "11000"}

	err := writeEOBDataToFile(&bbc, acoID, beneficiaryIDs)
	if err != nil {
		t.Fail()
	}

	filePath := fmt.Sprintf("%s/%s.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), acoID)
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var jsonOBJ map[string]interface{}
		err := json.Unmarshal(scanner.Bytes(), &jsonOBJ)
		assert.Nil(t, err)
		assert.NotNil(t, jsonOBJ["fullUrl"])
		assert.NotNil(t, jsonOBJ["resource"])
	}

	os.Remove(filePath)
}

func TestWriteEOBDataToFileNoClient(t *testing.T) {
	err := writeEOBDataToFile(nil, "9c05c1f8-349d-400f-9b69-7963f2262b08", []string{"20000", "21000"})
	assert.NotNil(t, err)
}

func TestWriteEOBDataToFileInvalidACO(t *testing.T) {
	bbc := MockBlueButtonClient{}
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262zzz"
	beneficiaryIDs := []string{"10000", "11000"}

	err := writeEOBDataToFile(&bbc, acoID, beneficiaryIDs)
	assert.NotNil(t, err)
}

func TestWriteEOBDataToFileWithError(t *testing.T) {
	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	bbc := MockBlueButtonClientWithError{}
	acoID := "387c3a62-96fa-4d93-a5d0-fd8725509dd9"
	beneficiaryIDs := []string{"10000", "11000"}

	err := writeEOBDataToFile(&bbc, acoID, beneficiaryIDs)
	if err != nil {
		t.Fail()
	}

	filePath := fmt.Sprintf("%s/%s-error.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), acoID)
	fData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fail()
	}

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 10000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 10000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 11000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 11000 in ACO 387c3a62-96fa-4d93-a5d0-fd8725509dd9"}}]}`
	assert.Equal(t, ooResp+"\n", string(fData))

	os.Remove(fmt.Sprintf("%s/%s.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), acoID))
	os.Remove(filePath)
}

func TestAppendErrorToFile(t *testing.T) {
	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	acoID := "328e83c3-bc46-4827-836c-0ba0c713dc7d"

	appendErrorToFile(acoID, "", "", "")

	filePath := fmt.Sprintf("%s/%s-error.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), acoID)
	fData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fail()
	}

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"Error"}]}`

	assert.Equal(t, ooResp+"\n", string(fData))

	os.Remove(filePath)
}

// Returns copy of a static json file (From Blue Button Sandbox originally) after replacing the patient ID of 20000000000001 with the requested identifier

func (bbc *MockBlueButtonClient) GetData(path string, params url.Values) (string, error) {
	var identifier string
	if strings.Contains(path, "Coverage") {
		identifier = "beneficiary"
	} else if strings.Contains(path, "Patient") {
		identifier = "_id"
	} else if strings.Contains(path, "ExplanationOfBenefit") {
		identifier = "patient"
	}

	fData, err := ioutil.ReadFile("../shared_files" + strings.TrimRight(path, "/"))

	// What other things should we be overwriting?  File creation time?
	cleanData := strings.Replace(string(fData), "20000000000001", params.Get(identifier), -1)

	return cleanData, err
}

func (bbc *MockBlueButtonClientWithError) GetData(path string, params url.Values) (string, error) {

	return "", errors.New("Error")
}

func (s *MainTestSuite) TestProcessJob() {
	db := database.GetGORMDbConnection()
	defer db.Close()

	j := models.Job{
		AcoID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
	}
	db.Save(&j)

	jobArgs := new(jobEnqueueArgs)
	jobArgs.ID = int(j.ID)
	jobArgs.AcoID = j.AcoID.String()
	jobArgs.UserID = j.UserID.String()
	jobArgs.BeneficiaryIDs = []string{"10000", "11000"}
	args, _ := json.Marshal(jobArgs)

	job := &que.Job{
		Type: "ProcessJob",
		Args: args,
	}
	fmt.Println("About to queue up the job")
	err := processJob(job)
	assert.Nil(s.T(), err)
	var completedJob models.Job
	err = db.First(&completedJob, "ID = ?", jobArgs.ID).Error
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "Completed", completedJob.Status)
}

func (s *MainTestSuite) TestSetupQueue() {
	var wp *pgx.ConnPool = setupQueue()
	wp.Close()
	os.Setenv("WORKER_POOL_SIZE", "7")
	wp = setupQueue()
	wp.Close()
}
