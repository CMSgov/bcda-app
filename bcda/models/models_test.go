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
	j := Job{Status: constants.InProgress, JobCount: 25}
	assert.Equal(s.T(), "In Progress (24%)", j.StatusMessage(6))

	j = Job{Status: constants.InProgress, JobCount: 0}
	assert.Equal(s.T(), constants.InProgress, j.StatusMessage(0))

	j = Job{Status: JobStatusCompleted, JobCount: 25}
	assert.Equal(s.T(), string(JobStatusCompleted), j.StatusMessage(25))
}

func (s *ModelsTestSuite) TestACODenylist() {
	pastCutoff := time.Now().Add(-24 * time.Hour)
	futureCutoff := time.Now().Add(24 * time.Hour)

	tests := []struct {
		title          string
		td             *Termination
		expectedResult bool
	}{
		{"Past Cutoff Involuntary", &Termination{CutoffDate: pastCutoff, DenylistType: Involuntary}, true},
		{"Past Cutoff Voluntary", &Termination{CutoffDate: pastCutoff, DenylistType: Voluntary}, true},
		{"Past Cutoff Limited", &Termination{CutoffDate: pastCutoff, DenylistType: Limited}, true},
		{"Future Cutoff Involuntary", &Termination{CutoffDate: futureCutoff, DenylistType: Involuntary}, false},
		{"Future Cutoff Voluntary", &Termination{CutoffDate: futureCutoff, DenylistType: Voluntary}, false},
		{"Future Cutoff Limited", &Termination{CutoffDate: futureCutoff, DenylistType: Limited}, false},
		{"Zero Cutoff Involuntary", &Termination{DenylistType: Involuntary}, true},
		{"Zero Cutoff Voluntary", &Termination{DenylistType: Voluntary}, true},
		{"Zero Cutoff Limited", &Termination{DenylistType: Limited}, false},
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
			assert.Equal(s.T(), tt.expectedResult, aco.Denylisted())
		})
	}
}
