package queueing

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
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
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

// These are set in que_test.go
// var logger = logrus.New()
// var logHook = test.NewLocal(logger)

// TestWork acts as an end-to-end verification of the entire process:
// adding a job, picking up the job, processing the job, and closing the client
func TestWork_Integration(t *testing.T) {
	// Set up the logger since we're using the real client
	client.SetLogger(logger)

	// Reset our environment once we've finished with the test
	defer func(origEnqueuer string) {
		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
	}(conf.GetEnv("QUEUE_LIBRARY"))

	conf.SetEnv(t, "QUEUE_LIBRARY", "river")

	defer func(payload, staging string) {
		conf.SetEnv(t, "FHIR_PAYLOAD_DIR", payload)
		conf.SetEnv(t, "FHIR_STAGING_DIR", staging)
	}(conf.GetEnv("FHIR_PAYLOAD_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))

	defer func(cert, key, ca string) {
		conf.SetEnv(t, "BB_CLIENT_CERT_FILE", cert)
		conf.SetEnv(t, "BB_CLIENT_KEY_FILE", key)
		conf.SetEnv(t, "BB_CLIENT_CA_FILE", ca)
	}(conf.GetEnv("BB_CLIENT_CERT_FILE"), conf.GetEnv("BB_CLIENT_KEY_FILE"), conf.GetEnv("BB_CLIENT_CA_FILE"))

	conf.SetEnv(t, "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(t, "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(t, "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")

	// Ensure we do not clutter our working directory with any data
	tempDir1, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDir2, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	conf.SetEnv(t, "FHIR_PAYLOAD_DIR", tempDir1)
	conf.SetEnv(t, "FHIR_STAGING_DIR", tempDir2)

	db := database.Connection

	cmsID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{UUID: uuid.NewRandom(), CMSID: &cmsID}
	postgrestest.CreateACO(t, db, aco)
	job := models.Job{ACOID: aco.UUID, Status: models.JobStatusPending}
	postgrestest.CreateJobs(t, db, &job)

	defer postgrestest.DeleteACO(t, db, aco.UUID)

	q := StartRiver(1)
	defer q.StopRiver()

	id, _ := safecast.ToInt(job.ID)
	jobArgs := models.JobEnqueueArgs{ID: id, ACOID: cmsID, BBBasePath: uuid.New()}

	enqueuer := NewEnqueuer()
	assert.NoError(t, enqueuer.AddJob(context.Background(), jobArgs, 1))

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Job not completed in alloted time.")
			return
		default:
			currentJob := postgrestest.GetJobByID(t, db, job.ID)
			// don't wait for a job if it has a terminal status
			if isTerminalStatus(currentJob.Status) {
				return
			}
			logger.Infof("Waiting on job to be completed. Current status %s.", job.Status)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestCleanupJobWorker_Work(t *testing.T) {
	// Set up the logger since we're using the real client
	client.SetLogger(logger)

	// Reset our environment once we've finished with the test
	defer func(origEnqueuer string) {
		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
	}(conf.GetEnv("QUEUE_LIBRARY"))

	conf.SetEnv(t, "QUEUE_LIBRARY", "river")

	defer func(payload, staging, archive string) {
		conf.SetEnv(t, "FHIR_PAYLOAD_DIR", payload)
		conf.SetEnv(t, "FHIR_STAGING_DIR", staging)
		conf.SetEnv(t, "FHIR_ARCHIVE_DIR", archive)
	}(conf.GetEnv("FHIR_PAYLOAD_DIR"), conf.GetEnv("FHIR_STAGING_DIR"), conf.GetEnv("FHIR_ARCHIVE_DIR"))

	// Ensure we do not clutter our working directory with any data
	tempDir1, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDir2, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDir3, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	conf.SetEnv(t, "FHIR_PAYLOAD_DIR", tempDir1)
	conf.SetEnv(t, "FHIR_STAGING_DIR", tempDir2)
	conf.SetEnv(t, "FHIR_ARCHIVE_DIR", tempDir3)

	cleanupJobWorker := &CleanupJobWorker{}
	rjob := &river.Job[CleanupJobArgs]{}

	ctx := context.Background()
	err = cleanupJobWorker.Work(ctx, rjob)
	assert.NoError(t, err)
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
