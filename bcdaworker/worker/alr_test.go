package worker

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Initial Setup
type AlrWorkerTestSuite struct {
	suite.Suite
	alrWorker AlrWorker
	db        *sql.DB
	jobArgs   models.JobAlrEnqueueArgs
}

// Initial Setup
func (s *AlrWorkerTestSuite) SetupSuite() {
	s.db = database.Connection
	s.alrWorker = NewAlrWorker(s.db)

	// Create synthetic Data
	// TODO: Replace this with Martin's testing strategy from #4239
	exMap := make(map[string]string)
	exMap["EnrollFlag1"] = "1"
	exMap["HCC_version"] = "V12"
	exMap["HCC_COL_1"] = "1"
	exMap["HCC_COL_2"] = "0"
	aco := "A1234"
	MBIs := []string{"abd123abd01", "abd123abd02"}
	timestamp := time.Now()
	timestamp2 := timestamp.Add(time.Hour * 24)
	dob1, _ := time.Parse("01/02/2006", "01/20/1950")
	dob2, _ := time.Parse("01/02/2006", "04/15/1950")
	alrs := []models.Alr{
		{
			ID:            1, // These are set manually for testing
			MetaKey:       1, // PostgreSQL should automatically make these
			BeneMBI:       MBIs[0],
			BeneHIC:       "1q2w3e4r5t6y",
			BeneFirstName: "John",
			BeneLastName:  "Smith",
			BeneSex:       "1",
			BeneDOB:       dob1,
			BeneDOD:       time.Time{},
			KeyValue:      exMap,
		},
		{
			ID:            2,
			MetaKey:       2,
			BeneMBI:       MBIs[1],
			BeneHIC:       "0p9o8i7u6y5t",
			BeneFirstName: "Melissa",
			BeneLastName:  "Jones",
			BeneSex:       "2",
			BeneDOB:       dob2,
			BeneDOD:       time.Time{},
			KeyValue:      exMap,
		},
	}
	ctx := context.Background()

	// Add Data into repo
	_ = s.alrWorker.AlrRepository.AddAlr(ctx, aco, timestamp, alrs[:1])
	_ = s.alrWorker.AlrRepository.AddAlr(ctx, aco, timestamp2, alrs[1:2])

	// Create JobArgs
	s.jobArgs = models.JobAlrEnqueueArgs{
		ID:         1,
		CMSID:      aco,
		MBIs:       MBIs,
		LowerBound: timestamp,
		UpperBound: timestamp2,
	}

	tempDir, err := ioutil.TempDir("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}

	s.alrWorker.StagingDir = tempDir
}

// Test NewAlrWorker returns a worker alrWorker
func (s *AlrWorkerTestSuite) TestNewAlrWorker() {
	newAlrWorker := NewAlrWorker(s.db)
	assert.NotEmpty(s.T(), newAlrWorker.StagingDir)
	assert.NotNil(s.T(), newAlrWorker.AlrRepository)
}

// Test ProcessAlrJob
func (s *AlrWorkerTestSuite) TestProcessAlrJob() {
	ctx := context.Background()
	err := s.alrWorker.ProcessAlrJob(ctx, s.jobArgs)
	// Check Job is processed with no errors
	assert.NoError(s.T(), err)
}

func TestAlrWorkerTestSuite(t *testing.T) {
	d := new(AlrWorkerTestSuite)
	suite.Run(t, d)
	t.Cleanup(func() { os.RemoveAll(d.alrWorker.StagingDir) })
}
