package cleanup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

type CleanupTestSuite struct {
	suite.Suite
	testApp *cli.App

	testACO models.ACO

	db *sql.DB
}

func (s *CleanupTestSuite) setupJobFile(modified time.Time, status models.JobStatus, rootPath string) (uint, *os.File) {
	j := models.Job{
		ACOID:      s.testACO.UUID,
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     status,
		UpdatedAt:  modified,
	}

	postgrestest.CreateJobs(s.T(), s.db, &j)

	path := fmt.Sprintf("%s/%d", rootPath, j.ID)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		s.T().Error(err)
	}
	jobFile, err := os.Create(fmt.Sprintf("%s/%s", path, "fake.ndjson"))
	if err != nil {
		s.T().Error(err)
	}
	defer jobFile.Close()

	return j.ID, jobFile
}

type MockCleanupJob struct {
	mock.Mock
}

type MockArchiveExpiring struct {
	mock.Mock
}

func (m *MockCleanupJob) CleanupJob(maxDate time.Time, currentStatus, newStatus models.JobStatus, rootDirsToClean ...string) error {
	args := m.Called(maxDate, currentStatus, newStatus, rootDirsToClean)
	return args.Error(0)
}

func (m *MockArchiveExpiring) ArchiveExpiring(maxDate time.Time) error {
	args := m.Called(maxDate)
	return args.Error(0)
}

func TestCleanupJobWorker_Work(t *testing.T) {
	// Set up the logger since we're using the real client
	var logger = logrus.New()
	client.SetLogger(logger)

	// Create mock objects
	mockCleanupJob := new(MockCleanupJob)
	mockArchiveExpiring := new(MockArchiveExpiring)

	const archivePath = "/path/to/archive"
	const stagingPath = "/path/to/staging"
	const payloadPath = "/path/to/payload"

	mockCleanupJob.On("CleanupJob", mock.AnythingOfType("time.Time"), models.JobStatusArchived, models.JobStatusExpired, []string{archivePath, stagingPath}).Return(nil)
	mockCleanupJob.On("CleanupJob", mock.AnythingOfType("time.Time"), models.JobStatusFailed, models.JobStatusFailedExpired, []string{stagingPath, payloadPath}).Return(nil)
	mockCleanupJob.On("CleanupJob", mock.AnythingOfType("time.Time"), models.JobStatusCancelled, models.JobStatusCancelledExpired, []string{stagingPath, payloadPath}).Return(nil)
	mockArchiveExpiring.On("ArchiveExpiring", mock.AnythingOfType("time.Time")).Return(nil)

	// Create a worker instance
	cleanupJobWorker := &CleanupJobWorker{
		cleanupJob:      mockCleanupJob.CleanupJob,
		archiveExpiring: mockArchiveExpiring.ArchiveExpiring,
	}

	// Create a mock river.Job
	mockJob := &river.Job[CleanupJobArgs]{
		Args: CleanupJobArgs{},
	}

	// Call the Work function
	err := cleanupJobWorker.Work(context.Background(), mockJob)

	// Assert that there was no error
	assert.NoError(t, err)

	// Assert that all expectations were met
	mockCleanupJob.AssertExpectations(t)
	mockArchiveExpiring.AssertExpectations(t)
}

func assertFileNotExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file %s should not be found", path)
}

func (s *CleanupTestSuite) TestArchiveExpiring() {
	assert := assert.New(s.T())

	// condition: no jobs exist
	args := []string{"bcda", constants.ArchJobFiles}
	err := s.testApp.Run(args)
	assert.Nil(err)

	// timestamp to ensure that the job gets archived (older than the default 24h window)
	t := time.Now().Add(-48 * time.Hour)
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusCompleted,
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", "../bcdaworker/data/test")
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", constants.TestArchivePath)

	path := fmt.Sprintf("%s/%d/", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	newpath := conf.GetEnv("FHIR_ARCHIVE_DIR")

	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	if _, err = os.Stat(newpath); os.IsNotExist(err) {
		err = os.MkdirAll(newpath, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	f, err := os.Create(fmt.Sprintf("%s/fake.ndjson", path))
	if err != nil {
		s.T().Error(err)
	}
	defer f.Close()

	// condition: normal execution
	// execute the test case from CLI
	args = []string{"bcda", constants.ArchJobFiles}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// check that the file has moved to the archive location
	expPath := fmt.Sprintf("%s/%d/fake.ndjson", conf.GetEnv("FHIR_ARCHIVE_DIR"), j.ID)
	_, err = os.ReadFile(expPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(expPath, "File not Found")

	testJob := postgrestest.GetJobByID(s.T(), s.db, j.ID)

	// check the status of the job
	assert.Equal(models.JobStatusArchived, testJob.Status)

	// clean up
	os.RemoveAll(conf.GetEnv("FHIR_ARCHIVE_DIR"))
}

func (s *CleanupTestSuite) TestArchiveExpiringWithoutPayloadDir() {
	assert := assert.New(s.T())

	// condition: no jobs exist
	args := []string{"bcda", constants.ArchJobFiles}
	err := s.testApp.Run(args)
	assert.Nil(err)

	// timestamp to ensure that the job gets archived (older than the default 24h window)
	t := time.Now().Add(-48 * time.Hour)
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusCompleted,
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	// condition: normal execution
	// execute the test case from CLI
	args = []string{"bcda", constants.ArchJobFiles}
	err = s.testApp.Run(args)
	assert.Nil(err)

	testJob := postgrestest.GetJobByID(s.T(), s.db, j.ID)

	// check the status of the job
	assert.Equal(models.JobStatusArchived, testJob.Status)

	// clean up
	os.RemoveAll(conf.GetEnv("FHIR_ARCHIVE_DIR"))
}

func (s *CleanupTestSuite) TestArchiveExpiringWithThreshold() {
	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", "../bcdaworker/data/test")
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", constants.TestArchivePath)

	path := fmt.Sprintf("%s/%d/", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	f, err := os.Create(fmt.Sprintf("%s/fake.ndjson", path))
	if err != nil {
		s.T().Error(err)
	}
	defer f.Close()

	// execute the test case from CLI
	args := []string{"bcda", constants.ArchJobFiles, constants.ThresholdArg, "1"}
	err = s.testApp.Run(args)
	assert.Nil(s.T(), err)

	// check that the file has not moved to the archive location
	dataPath := fmt.Sprintf("%s/%d/fake.ndjson", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	_, err = os.ReadFile(dataPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(s.T(), dataPath, "File not Found")

	testJob := postgrestest.GetJobByID(s.T(), s.db, j.ID)
	// check the status of the job
	assert.Equal(s.T(), models.JobStatusCompleted, testJob.Status)

	// clean up
	os.Remove(dataPath)
}

func (s *CleanupTestSuite) TestCleanArchive() {
	// init
	const Threshold = 30
	now := time.Now()

	assert := assert.New(s.T())

	// condition: FHIR_ARCHIVE_DIR doesn't exist
	conf.UnsetEnv(s.T(), "FHIR_ARCHIVE_DIR")
	args := []string{"bcda", constants.CleanupArchArg, constants.ThresholdArg, strconv.Itoa(Threshold)}
	err := s.testApp.Run(args)
	assert.Nil(err)
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", constants.TestArchivePath)

	// condition: FHIR_STAGING_DIR doesn't exist
	conf.UnsetEnv(s.T(), "FHIR_STAGING_DIR")
	args = []string{"bcda", constants.CleanupArchArg, constants.ThresholdArg, strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)
	conf.SetEnv(s.T(), "FHIR_STAGING_DIR", constants.TestStagingPath)

	// condition: no jobs exist
	args = []string{"bcda", constants.CleanupArchArg, constants.ThresholdArg, strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// create a file that was last modified before the Threshold, but accessed after it
	modified := now.Add(-(time.Hour * (Threshold + 1)))
	beforeJobID, before := s.setupJobFile(modified, models.JobStatusArchived, conf.GetEnv("FHIR_ARCHIVE_DIR"))
	defer before.Close()

	// create a file that is clearly after the threshold (unless the threshold is 0)
	afterJobID, after := s.setupJobFile(now, models.JobStatusArchived, conf.GetEnv("FHIR_ARCHIVE_DIR"))
	defer after.Close()

	// condition: bad threshold value
	args = []string{"bcda", constants.CleanupArchArg, constants.ThresholdArg, "abcde"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid value \"abcde\" for flag -threshold: parse error")

	// condition: before < Threshold < after <= now
	// a file created before the Threshold should be deleted; one created after should not
	// we use last modified as a proxy for created, because these files should not be changed after creation
	args = []string{"bcda", constants.CleanupArchArg, constants.ThresholdArg, strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	assertFileNotExists(s.T(), before.Name())

	beforeJob := postgrestest.GetJobByID(s.T(), s.db, beforeJobID)
	assert.Equal(models.JobStatusExpired, beforeJob.Status)

	assert.FileExists(after.Name(), "%s not found; it should have been", after.Name())

	afterJob := postgrestest.GetJobByID(s.T(), s.db, afterJobID)
	assert.Equal(models.JobStatusArchived, afterJob.Status)

	// I think this is an application directory and should always exist, but that doesn't seem to be the norm
	os.RemoveAll(conf.GetEnv("FHIR_ARCHIVE_DIR"))
}

func (s *CleanupTestSuite) TestCleanupFailed() {
	const threshold = 30
	modified := time.Now().Add(-(time.Hour * (threshold + 1)))
	beforePayloadJobID, beforePayload := s.setupJobFile(modified, models.JobStatusFailed, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	beforeStagingJobID, beforeStaging := s.setupJobFile(modified, models.JobStatusFailed, conf.GetEnv("FHIR_STAGING_DIR"))
	// Job is old enough, but does not match the status
	completedJobID, completed := s.setupJobFile(modified, models.JobStatusCompleted, conf.GetEnv("FHIR_PAYLOAD_DIR"))

	afterPayloadJobID, afterPayload := s.setupJobFile(time.Now(), models.JobStatusFailed, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	afterStagingJobID, afterStaging := s.setupJobFile(time.Now(), models.JobStatusFailed, conf.GetEnv("FHIR_STAGING_DIR"))

	// Check that we can clean up jobs that do not have data
	noDataID, noData := s.setupJobFile(modified, models.JobStatusFailed, conf.GetEnv("FHIR_STAGING_DIR"))
	dir, _ := path.Split(noData.Name())
	os.RemoveAll(dir)
	assertFileNotExists(s.T(), noData.Name())

	shouldExist := []*os.File{afterPayload, afterStaging, completed}
	shouldNotExist := []*os.File{beforePayload, beforeStaging, noData}

	defer func() {
		for _, f := range append(shouldExist, shouldNotExist...) {
			os.Remove(f.Name())
			f.Close()
		}
	}()

	err := s.testApp.Run([]string{"bcda", "cleanup-failed", constants.ThresholdArg, strconv.Itoa(threshold)})
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), models.JobStatusFailedExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforePayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusFailedExpired,
		postgrestest.GetJobByID(s.T(), s.db, noDataID).Status)
	assert.Equal(s.T(), models.JobStatusFailedExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforeStagingJobID).Status)

	assert.Equal(s.T(), models.JobStatusFailed,
		postgrestest.GetJobByID(s.T(), s.db, afterPayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusFailed,
		postgrestest.GetJobByID(s.T(), s.db, afterStagingJobID).Status)
	assert.Equal(s.T(), models.JobStatusCompleted,
		postgrestest.GetJobByID(s.T(), s.db, completedJobID).Status)

	for _, f := range shouldExist {
		assert.FileExists(s.T(), f.Name())
	}
	for _, f := range shouldNotExist {
		assertFileNotExists(s.T(), f.Name())
	}
}

func (s *CleanupTestSuite) TestCleanupCancelled() {
	const threshold = 30
	modified := time.Now().Add(-(time.Hour * (threshold + 1)))
	beforePayloadJobID, beforePayload := s.setupJobFile(modified, models.JobStatusCancelled, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	beforeStagingJobID, beforeStaging := s.setupJobFile(modified, models.JobStatusCancelled, conf.GetEnv("FHIR_STAGING_DIR"))
	// Job is old enough, but does not match the status
	completedJobID, completed := s.setupJobFile(modified, models.JobStatusCompleted, conf.GetEnv("FHIR_PAYLOAD_DIR"))

	afterPayloadJobID, afterPayload := s.setupJobFile(time.Now(), models.JobStatusCancelled, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	afterStagingJobID, afterStaging := s.setupJobFile(time.Now(), models.JobStatusCancelled, conf.GetEnv("FHIR_STAGING_DIR"))

	// Check that we can clean up jobs that do not have data
	noDataID, noData := s.setupJobFile(modified, models.JobStatusCancelled, conf.GetEnv("FHIR_STAGING_DIR"))
	dir, _ := path.Split(noData.Name())
	os.RemoveAll(dir)
	assertFileNotExists(s.T(), noData.Name())

	shouldExist := []*os.File{afterPayload, afterStaging, completed}
	shouldNotExist := []*os.File{beforePayload, beforeStaging, noData}

	defer func() {
		for _, f := range append(shouldExist, shouldNotExist...) {
			os.Remove(f.Name())
			f.Close()
		}
	}()

	err := s.testApp.Run([]string{"bcda", "cleanup-cancelled", constants.ThresholdArg, strconv.Itoa(threshold)})
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), models.JobStatusCancelledExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforePayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusCancelledExpired,
		postgrestest.GetJobByID(s.T(), s.db, noDataID).Status)
	assert.Equal(s.T(), models.JobStatusCancelledExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforeStagingJobID).Status)

	assert.Equal(s.T(), models.JobStatusCancelled,
		postgrestest.GetJobByID(s.T(), s.db, afterPayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusCancelled,
		postgrestest.GetJobByID(s.T(), s.db, afterStagingJobID).Status)
	assert.Equal(s.T(), models.JobStatusCompleted,
		postgrestest.GetJobByID(s.T(), s.db, completedJobID).Status)

	for _, f := range shouldExist {
		assert.FileExists(s.T(), f.Name())
	}
	for _, f := range shouldNotExist {
		assertFileNotExists(s.T(), f.Name())
	}
}
