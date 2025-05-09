package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"

	"github.com/stretchr/testify/mock"
)

type MockBlueButtonClient struct {
	mock.Mock
	HICN *string
	MBI  *string
}

func (bbc *MockBlueButtonClient) GetExplanationOfBenefit(jobData worker_types.JobEnqueueArgs, patientID string, serviceDate ClaimsWindow) (*fhirModels.Bundle, error) {
	args := bbc.Called(jobData, patientID, serviceDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*fhirModels.Bundle), args.Error(1)
}

func (bbc *MockBlueButtonClient) GetPatientByMbi(jobData worker_types.JobEnqueueArgs, mbi string) (string, error) {
	args := bbc.Called(mbi)
	return args.String(0), args.Error(1)
}

func (bbc *MockBlueButtonClient) GetPatient(jobData worker_types.JobEnqueueArgs, patientID string) (*fhirModels.Bundle, error) {
	args := bbc.Called(jobData, patientID)
	return args.Get(0).(*fhirModels.Bundle), args.Error(1)
}

func (bbc *MockBlueButtonClient) GetCoverage(jobData worker_types.JobEnqueueArgs, beneficiaryID string) (*fhirModels.Bundle, error) {
	args := bbc.Called(jobData, beneficiaryID)
	return args.Get(0).(*fhirModels.Bundle), args.Error(1)
}

func (bbc *MockBlueButtonClient) GetClaim(jobData worker_types.JobEnqueueArgs, mbi string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error) {
	args := bbc.Called(jobData, mbi, claimsWindow)
	return args.Get(0).(*fhirModels.Bundle), args.Error(1)
}

func (bbc *MockBlueButtonClient) GetClaimResponse(jobData worker_types.JobEnqueueArgs, mbi string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error) {
	args := bbc.Called(jobData, mbi, claimsWindow)
	return args.Get(0).(*fhirModels.Bundle), args.Error(1)
}

// Returns copy of a static json file (From Blue Button Sandbox originally) after replacing the patient ID of 20000000000001 with the requested identifier
// This is private in the real function and should remain so, but in the test client it makes maintenance easier to expose it.
func (bbc *MockBlueButtonClient) GetData(endpoint, patientID string) (string, error) {
	var fData []byte
	fData, err := os.ReadFile(filepath.Join("../shared_files/synthetic_beneficiary_data/", filepath.Clean(endpoint)))
	if err != nil {
		fData, err = os.ReadFile(filepath.Join("../../shared_files/synthetic_beneficiary_data/", filepath.Clean(endpoint)))
		if err != nil {
			return "", err
		}
	}
	cleanData := strings.ReplaceAll(string(fData), "20000000000001", patientID)
	if bbc.MBI != nil {
		// no longer hashed, but this is only a test file with synthetic test data
		cleanData = strings.ReplaceAll(cleanData, "-1Q03Z002871", *bbc.MBI)
	}
	return cleanData, err
}

func (bbc *MockBlueButtonClient) GetBundleData(endpoint, patientID string) (*fhirModels.Bundle, error) {
	payload, err := bbc.GetData(endpoint, patientID)
	if err != nil {
		return nil, err
	}

	var b fhirModels.Bundle
	err = json.Unmarshal([]byte(payload), &b)
	if err != nil {
		return nil, err
	}

	return &b, err
}
