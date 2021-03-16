package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
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
	db, _ := databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
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
	assert := assert.New(s.T())
	s.svc.(*service).alrMBIsPerJob = 2
	timeWindow := AlrRequestWindow{LowerBound: time.Now().Add(-24 * time.Hour), UpperBound: time.Now()}
	jobs, err := s.svc.GetAlrJobs(context.Background(), s.cmsID, RunoutAlrRequest, timeWindow)
	assert.NoError(err)
	// See testdata/cclf_beneficiaries.yml for information about the number of benes/MBIs
	// Should have 5 total benes for the runout job
	assert.Len(jobs, 3)
	var mbis []string
	for _, job := range jobs {
		assert.Equal(s.cmsID, job.ACO)
		assert.True(timeWindow.LowerBound.Equal(job.LowerBound))
		assert.True(timeWindow.UpperBound.Equal(job.UpperBound))
		mbis = append(mbis, job.MBIs...)
	}
	assert.Len(mbis, 5)
	for i := 0; i < 5; i++ {
		assert.Contains(mbis, fmt.Sprintf("ALR_RUN_%03d", i))
	}
}

func (s *AlrTestSuite) TestDefaultRequest() {
	assert := assert.New(s.T())
	s.svc.(*service).alrMBIsPerJob = 3
	timeWindow := AlrRequestWindow{LowerBound: time.Now().Add(-24 * time.Hour), UpperBound: time.Now()}
	jobs, err := s.svc.GetAlrJobs(context.Background(), s.cmsID, DefaultAlrRequest, timeWindow)
	assert.NoError(err)
	// See testdata/cclf_beneficiaries.yml for information about the number of benes/MBIs
	// Should have 4 total benes for the default job
	assert.Len(jobs, 2)
	var mbis []string
	for _, job := range jobs {
		assert.Equal(s.cmsID, job.ACO)
		assert.True(timeWindow.LowerBound.Equal(job.LowerBound))
		assert.True(timeWindow.UpperBound.Equal(job.UpperBound))
		mbis = append(mbis, job.MBIs...)
	}
	assert.Len(mbis, 4)
	for i := 0; i < 4; i++ {
		assert.Contains(mbis, fmt.Sprintf("ALR_REG_%03d", i))
	}
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
			assert.Equal(t, tt.expNumParts, numParts)
			assert.Equal(t, benes, total)
		})
	}
}
