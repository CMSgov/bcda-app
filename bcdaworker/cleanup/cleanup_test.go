package cleanup

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CleanupTestSuite struct {
	suite.Suite
	testACO            models.ACO
	db                 *sql.DB
	pendingDeletionDir string
}

func TestCleanupTestSuite(t *testing.T) {
	suite.Run(t, new(CleanupTestSuite))
}

func (s *CleanupTestSuite) SetupSuite() {
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(&s.Suite, dir)

	s.db = database.Connect()

	cmsID := testUtils.RandomHexID()[0:4]
	s.testACO = models.ACO{Name: uuid.New(), UUID: uuid.NewRandom(), ClientID: uuid.New(), CMSID: &cmsID}
	postgrestest.CreateACO(s.T(), s.db, s.testACO)
}

func (s *CleanupTestSuite) TearDownTest() {
	os.RemoveAll(s.pendingDeletionDir)
	testUtils.PrintSeparator()
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

func assertFileNotExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file %s should not be found", path)
}

func (s *CleanupTestSuite) TestArchiveExpiring() {
	assert := assert.New(s.T())

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

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	if _, err := os.Stat(newpath); os.IsNotExist(err) {
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

	if err := ArchiveExpiring(s.db, t); err != nil {
		s.T().Error(err)
	}

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
	// Remove Payload directory
	os.RemoveAll(conf.GetEnv("FHIR_PAYLOAD_DIR"))
	assert := assert.New(s.T())

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

	if err := ArchiveExpiring(s.db, t); err != nil {
		s.T().Error(err)
	}

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

	payload := fmt.Sprintf("%s/%d/", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	if _, err := os.Stat(payload); os.IsNotExist(err) {
		err = os.MkdirAll(payload, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	f, err := os.Create(fmt.Sprintf("%s/fake.ndjson", payload))
	if err != nil {
		s.T().Error(err)
	}
	defer f.Close()

	if err := ArchiveExpiring(s.db, time.Now().Add(-24*time.Hour)); err != nil {
		s.T().Error(err)
	}

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
	err := CleanupJob(s.db, now.Add(-Threshold*time.Hour), models.JobStatusArchived, models.JobStatusExpired,
		conf.GetEnv("FHIR_ARCHIVE_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))
	assert.Nil(err)
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", constants.TestArchivePath)

	// condition: FHIR_STAGING_DIR doesn't exist
	conf.UnsetEnv(s.T(), "FHIR_STAGING_DIR")
	err = CleanupJob(s.db, now.Add(-Threshold*time.Hour), models.JobStatusArchived, models.JobStatusExpired,
		conf.GetEnv("FHIR_ARCHIVE_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))
	assert.Nil(err)
	conf.SetEnv(s.T(), "FHIR_STAGING_DIR", constants.TestStagingPath)

	// // condition: no jobs exist
	err = CleanupJob(s.db, now.Add(-Threshold*time.Hour), models.JobStatusArchived, models.JobStatusExpired,
		conf.GetEnv("FHIR_ARCHIVE_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))
	if err != nil {
		s.T().Error(err)
	}

	// create a file that was last modified before the Threshold, but accessed after it
	modified := now.Add(-(time.Hour * (Threshold + 1)))
	beforeJobID, before := s.setupJobFile(modified, models.JobStatusArchived, conf.GetEnv("FHIR_ARCHIVE_DIR"))
	defer before.Close()

	// create a file that is clearly after the threshold (unless the threshold is 0)
	afterJobID, after := s.setupJobFile(now, models.JobStatusArchived, conf.GetEnv("FHIR_ARCHIVE_DIR"))
	defer after.Close()

	// condition: before < Threshold < after <= now
	// a file created before the Threshold should be deleted; one created after should not
	// we use last modified as a proxy for created, because these files should not be changed after creation
	err = CleanupJob(s.db, now.Add(-Threshold*time.Hour), models.JobStatusArchived, models.JobStatusExpired,
		conf.GetEnv("FHIR_ARCHIVE_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))
	assert.Nil(err)
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
	payload := conf.GetEnv("FHIR_PAYLOAD_DIR")
	staging := conf.GetEnv("FHIR_STAGING_DIR")

	beforePayloadJobID, beforePayload := s.setupJobFile(modified, models.JobStatusFailed, payload)
	beforeStagingJobID, beforeStaging := s.setupJobFile(modified, models.JobStatusFailed, staging)
	// Job is old enough, but does not match the status
	completedJobID, completed := s.setupJobFile(modified, models.JobStatusCompleted, payload)

	afterPayloadJobID, afterPayload := s.setupJobFile(time.Now(), models.JobStatusFailed, payload)
	afterStagingJobID, afterStaging := s.setupJobFile(time.Now(), models.JobStatusFailed, staging)

	// Check that we can clean up jobs that do not have data
	noDataID, noData := s.setupJobFile(modified, models.JobStatusFailed, staging)
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

	err := CleanupJob(s.db, time.Now().Add(-threshold*time.Hour), models.JobStatusFailed, models.JobStatusFailedExpired,
		staging, payload)
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
	payload := conf.GetEnv("FHIR_PAYLOAD_DIR")
	staging := conf.GetEnv("FHIR_STAGING_DIR")

	beforePayloadJobID, beforePayload := s.setupJobFile(modified, models.JobStatusCancelled, payload)
	beforeStagingJobID, beforeStaging := s.setupJobFile(modified, models.JobStatusCancelled, staging)

	// Job is old enough, but does not match the status
	completedJobID, completed := s.setupJobFile(modified, models.JobStatusCompleted, payload)

	afterPayloadJobID, afterPayload := s.setupJobFile(time.Now(), models.JobStatusCancelled, payload)
	afterStagingJobID, afterStaging := s.setupJobFile(time.Now(), models.JobStatusCancelled, staging)

	// Check that we can clean up jobs that do not have data
	noDataID, noData := s.setupJobFile(modified, models.JobStatusCancelled, staging)

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

	err := CleanupJob(s.db, modified, models.JobStatusCancelled, models.JobStatusCancelledExpired,
		staging, payload)
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
