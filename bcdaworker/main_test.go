package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var eobData = `{
	"resourceType": "Bundle",
	"id": "c6a3f338-5f25-4001-9b8f-023d9297c6ce",
	"meta": {
	  "lastUpdated": "2018-10-01T17:02:22.544-04:00"
	},
	"type": "searchset",
	"total": 1,
	"link": [],
	"entry": []
}`

type MockBlueButtonClient struct {
	mock.Mock
}

type MockBlueButtonClientWithError struct {
	mock.Mock
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
	fData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fail()
	}

	assert.Equal(t, eobData+"\n"+eobData+"\n", string(fData))

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
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262b07"
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

	ooResp := `{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 10000 in ACO 9c05c1f8-349d-400f-9b69-7963f2262b07"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 10000 in ACO 9c05c1f8-349d-400f-9b69-7963f2262b07"}}]}
{"resourceType":"OperationOutcome","issue":[{"severity":"Error","code":"Exception","details":{"coding":[{"display":"Error retrieving ExplanationOfBenefit for beneficiary 11000 in ACO 9c05c1f8-349d-400f-9b69-7963f2262b07"}],"text":"Error retrieving ExplanationOfBenefit for beneficiary 11000 in ACO 9c05c1f8-349d-400f-9b69-7963f2262b07"}}]}`
	assert.Equal(t, ooResp+"\n", string(fData))

	os.Remove(filePath)
}

func TestAppendErrorToFile(t *testing.T) {
	os.Setenv("FHIR_PAYLOAD_DIR", "data/test")
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262b07"

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

func (bbc *MockBlueButtonClient) GetExplanationOfBenefitData(patientID string) (string, error) {
	return eobData, nil
}

func (bbc *MockBlueButtonClientWithError) GetExplanationOfBenefitData(patientID string) (string, error) {
	return "", errors.New("Error")
}
