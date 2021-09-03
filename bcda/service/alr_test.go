package service

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AlrTestSuite struct {
	suite.Suite
	cmsID string
	svc   Service
	r     models.Repository
}

func TestAlrTestSuite(t *testing.T) {
	suite.Run(t, new(AlrTestSuite))
}

func (s *AlrTestSuite) SetupSuite() {
	s.cmsID = "A9994"
	db := database.Connection // db-unit-test

	r := postgres.NewRepository(db)
	cfg, err := LoadConfig()
	if err != nil {
		assert.FailNowf(s.T(), "Failed ot load service config", err.Error())
	}

	// Test case here is 54, so settting it to 10
	cfg.AlrJobSize = 10

	s.svc = NewService(r, cfg, "/v1/fhir")
	s.r = r
}

func (s *AlrTestSuite) TestGetAlrJob() {
	ctx := context.Background()
	alrMBIs, err := s.r.GetAlrMBIs(ctx, s.cmsID)
	s.NoError(err)
	alrJobs := s.svc.GetAlrJobs(ctx, alrMBIs)
	// There should be 54 benes split into groups of max 10
	// Therefore should be a length of 6
	s.Equal(6, len(alrJobs))
}

// Not being used after refactor. TODO: delete
func TestPartitionBenes(t *testing.T) {
	var benes []*models.CCLFBeneficiary
	for i := 0; i < 15; i++ {
		benes = append(benes, &models.CCLFBeneficiary{MBI: testUtils.RandomMBI(t)})
	}

	tests := []struct {
		name        string
		size        uint
		expNumParts int
	}{
		{"InputEqualParts", 3, 5},
		{"InputNotEqual", 4, 4},
		{"SizeLargerThanInput", 30, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				total    []*models.CCLFBeneficiary
				part     []*models.CCLFBeneficiary
				start    = benes
				numParts int
			)
			for {
				part, start = partitionBenes(start, tt.size)
				if len(part) == 0 {
					break
				}
				numParts++
				total = append(total, part...)
			}
			assert.Equal(t, tt.expNumParts, numParts)
			assert.Equal(t, benes, total)
		})
	}
}
