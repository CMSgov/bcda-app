package worker

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Initial Setup
type AlrWorkerTestSuite struct {
	suite.Suite
	alrWorker AlrWorker
	db        *sql.DB
	jobArgs   []models.JobAlrEnqueueArgs
}

// Initial Setup
func (s *AlrWorkerTestSuite) SetupSuite() {
	s.db = database.Connection
	s.alrWorker = NewAlrWorker(s.db)

	r := postgres.NewRepository(s.db)
	MBIs, err := r.GetAlrMBIs(context.Background(), "A9994")
	s.NoError(err)

	// Test V1
	s.jobArgs = []models.JobAlrEnqueueArgs{{
		CMSID:           MBIs.CMSID,
		MBIs:            MBIs.MBIS,
		MetaKey:         MBIs.Metakey,
		BBBasePath:      "/v1/fhir",
		TransactionTime: MBIs.TransactionTime,
	}}
	// Test V2
	s.jobArgs = append(s.jobArgs, models.JobAlrEnqueueArgs{
		CMSID:           MBIs.CMSID,
		MBIs:            MBIs.MBIS,
		MetaKey:         MBIs.Metakey,
		BBBasePath:      "/v2/fhir",
		TransactionTime: MBIs.TransactionTime,
	})

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
	err := s.alrWorker.ProcessAlrJob(ctx, s.jobArgs[0])
	// Check Job is processed with no errors
	assert.NoError(s.T(), err)
	err = s.alrWorker.ProcessAlrJob(ctx, s.jobArgs[1])
	// Check Job is processed with no errors
	assert.NoError(s.T(), err)
}

func TestAlrWorkerTestSuite(t *testing.T) {
	d := new(AlrWorkerTestSuite)
	suite.Run(t, d)
	t.Cleanup(func() { os.RemoveAll(d.alrWorker.StagingDir) })
}
