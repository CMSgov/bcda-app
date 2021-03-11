package service

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AlrTestSuite struct {
	suite.Suite
	cmsID string
	svc   Service
}

func TestAlrTestSuite(t *testing.T) {
	suite.Run(t, new(AlrTestSuite))
}

func (s *AlrTestSuite) SetupSuite() {
	s.cmsID = "A0001" // See testdata/acos.yml
	db, _ := database.CreateDatabase(s.T(), true)
	tf, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)
	
	if err != nil {
		assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
	}
	if err := tf.Load(); err != nil {
		assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
	}
	r := postgres.NewRepository(db)
	cfg, err := LoadConfig()
	if err != nil {
		assert.FailNowf(s.T(), "Failed ot load service config", err.Error())
	}

	s.svc = NewService(r, cfg, "v1")
}

func (s *AlrTestSuite) TestRunoutRequest() {
	jobs, err := s.svc.GetAlrJobs(context.Background(), s.cmsID, RunoutAlrRequest, AlrRequestWindow{})
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), jobs)
}

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
			assert.Equal(t, benes, total)
		})
	}
}
