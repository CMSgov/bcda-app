package models

import (
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/pborman/uuid"
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

func (s *ModelsTestSuite) TestACOBlacklist() {
	blackListDate := time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local)
	blackListValues := []Termination{
		{
			TerminationDate: blackListDate,
			CutoffDate:      blackListDate,
			BlacklistType:   Involuntary,
		},
		{
			TerminationDate: blackListDate,
			CutoffDate:      blackListDate,
			BlacklistType:   Voluntary,
		},
		{
			TerminationDate: blackListDate,
			CutoffDate:      blackListDate,
			BlacklistType:   Limited,
		},
	}
	tests := []struct {
		title          string
		td             *Termination
		expectedResult bool
	}{
		{"Details Involuntary", &blackListValues[0], true},
		{"Details Voluntary", &blackListValues[1], true},
		{"Details Limited", &blackListValues[2], false},
		{"Details Missing", &Termination{}, true},
		{"Null", nil, false},
	}

	for _, tt := range tests {
		s.T().Run(string(tt.title), func(t *testing.T) {
			cmsID := testUtils.RandomHexID()[0:4]
			aco := ACO{
				UUID:               uuid.NewUUID(),
				CMSID:              &cmsID,
				Name:               "Blacklisted ACO",
				TerminationDetails: tt.td,
			}
			assert.Equal(s.T(), aco.Blacklisted(), tt.expectedResult)
		})
	}
}
