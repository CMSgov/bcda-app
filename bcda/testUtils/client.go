package testUtils

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	models "github.com/CMSgov/bcda-app/bcda/models/fhir"

	"github.com/stretchr/testify/mock"
)

type BlueButtonClient struct {
	mock.Mock
	HICN *string
	MBI  *string
}

func (bbc *BlueButtonClient) GetExplanationOfBenefit(patientID, jobID, cmsID, since string, transactionTime, serviceDate time.Time) (*models.Bundle, error) {
	args := bbc.Called(patientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bundle), args.Error(1)
}

func (bbc *BlueButtonClient) GetPatientByIdentifierHash(hashedIdentifier string) (string, error) {
	args := bbc.Called(hashedIdentifier)
	return args.String(0), args.Error(1)
}

func (bbc *BlueButtonClient) GetPatient(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	args := bbc.Called(patientID)
	return args.Get(0).(*models.Bundle), args.Error(1)
}

func (bbc *BlueButtonClient) GetCoverage(beneficiaryID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	args := bbc.Called(beneficiaryID)
	return args.Get(0).(*models.Bundle), args.Error(1)
}

// Returns copy of a static json file (From Blue Button Sandbox originally) after replacing the patient ID of 20000000000001 with the requested identifier
// This is private in the real function and should remain so, but in the test client it makes maintenance easier to expose it.
func (bbc *BlueButtonClient) GetData(endpoint, patientID string) (string, error) {
	var fData []byte
	fData, err := ioutil.ReadFile(filepath.Join("../shared_files/synthetic_beneficiary_data/", filepath.Clean(endpoint)))
	if err != nil {
		fData, err = ioutil.ReadFile(filepath.Join("../../shared_files/synthetic_beneficiary_data/", filepath.Clean(endpoint)))
		if err != nil {
			return "", err
		}
	}
	cleanData := strings.Replace(string(fData), "20000000000001", patientID, -1)
	if bbc.MBI != nil {
		// no longer hashed, but this is only a test file with synthetic test data
		cleanData = strings.Replace(cleanData, "-1Q03Z002871", *bbc.MBI, -1)
	}
	return cleanData, err
}

func (bbc *BlueButtonClient) GetBundleData(endpoint, patientID string) (*models.Bundle, error) {
	payload, err := bbc.GetData(endpoint, patientID)
	if err != nil {
		return nil, err
	}

	var b models.Bundle
	err = json.Unmarshal([]byte(payload), &b)
	if err != nil {
		return nil, err
	}

	return &b, err
}
