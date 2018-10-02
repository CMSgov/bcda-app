package main

import (
	"fmt"
	"io/ioutil"
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

func TestWriteEOBDataToFile(t *testing.T) {
	bbc := MockBlueButtonClient{}
	acoID := "9c05c1f8-349d-400f-9b69-7963f2262b07"
	beneficiaryIDs := []string{"10000", "11000"}

	err := writeEOBDataToFile(&bbc, acoID, beneficiaryIDs)
	if err != nil {
		t.Fail()
	}

	fData, err := ioutil.ReadFile(fmt.Sprintf("data/%s.ndjson", acoID))
	if err != nil {
		t.Fail()
	}

	assert.Equal(t, eobData+"\n", string(fData))
}

func (bbc *MockBlueButtonClient) GetExplanationOfBenefitData(patientID string) (string, error) {
	return eobData, nil
}
