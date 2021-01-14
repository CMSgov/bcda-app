package models

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (s *ModelsTestSuite) TestJobStatusMessage() {
	j := Job{Status: "In Progress", JobCount: 25, CompletedJobCount: 6}
	assert.Equal(s.T(), "In Progress (24%)", j.StatusMessage())

	j = Job{Status: "In Progress", JobCount: 0, CompletedJobCount: 0}
	assert.Equal(s.T(), "In Progress", j.StatusMessage())

	j = Job{Status: JobStatusCompleted, JobCount: 25, CompletedJobCount: 25}
	assert.Equal(s.T(), string(JobStatusCompleted), j.StatusMessage())
}

func (s *ModelsTestSuite) TestGetBlueButtonID_CCLFBeneficiary() {
	assert := s.Assert()
	cclfBeneficiary := CCLFBeneficiary{MBI: "MBI"}
	bbc := testUtils.BlueButtonClient{}
	bbc.MBI = &cclfBeneficiary.MBI

	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", "BB_VALUE"))

	cclfBeneficiary.BlueButtonID = ""
	// New never seen before mbi, asks the mock blue button client for the value
	blueButtonID, err := cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// The object has a BB ID set on it already, but we still ask mock blue button client for the value
	// We should receive the BB_VALUE since we are ignoring cached values
	cclfBeneficiary.BlueButtonID = "LOCAL_VAL"
	blueButtonID, err = cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// Should be making two calls to BB for the MBI_MODE attemptsm, but this number will be four with the earlier test in this method.
	// This is due to the fact that we are not relying on cached identifiers
	bbc.AssertNumberOfCalls(s.T(), "GetPatientByIdentifierHash", 2)
}
