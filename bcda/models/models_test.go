package models

import (
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
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
	j := Job{Status: constants.InProgress, JobCount: 25, CompletedJobCount: 6}
	assert.Equal(s.T(), "In Progress (24%)", j.StatusMessage())

	j = Job{Status: constants.InProgress, JobCount: 0, CompletedJobCount: 0}
	assert.Equal(s.T(), constants.InProgress, j.StatusMessage())

	j = Job{Status: JobStatusCompleted, JobCount: 25, CompletedJobCount: 25}
	assert.Equal(s.T(), string(JobStatusCompleted), j.StatusMessage())
}

func (s *ModelsTestSuite) TestACODenylist() {
	denyListDate := time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local)
	denyListValues := []Termination{
		{
			TerminationDate: denyListDate,
			CutoffDate:      denyListDate,
			DenylistType:    Involuntary,
		},
		{
			TerminationDate: denyListDate,
			CutoffDate:      denyListDate,
			DenylistType:    Voluntary,
		},
		{
			TerminationDate: denyListDate,
			CutoffDate:      denyListDate,
			DenylistType:    Limited,
		},
	}
	tests := []struct {
		title          string
		td             *Termination
		expectedResult bool
	}{
		{"Details Involuntary", &denyListValues[0], true},
		{"Details Voluntary", &denyListValues[1], true},
		{"Details Limited", &denyListValues[2], false},
		{"Details Missing", &Termination{}, true},
		{"Null", nil, false},
	}

	for _, tt := range tests {
		s.T().Run(string(tt.title), func(t *testing.T) {
			cmsID := testUtils.RandomHexID()[0:4]
			aco := ACO{
				UUID:               uuid.NewUUID(),
				CMSID:              &cmsID,
				Name:               "Denylisted ACO",
				TerminationDetails: tt.td,
			}
			assert.Equal(s.T(), aco.Denylisted(), tt.expectedResult)
		})
	}
}
