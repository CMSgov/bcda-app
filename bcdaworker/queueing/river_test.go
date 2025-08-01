package queueing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var logger = logrus.New()

func isTerminalStatus(status models.JobStatus) bool {
	switch status {
	case models.JobStatusCompleted,
		models.JobStatusCancelled,
		models.JobStatusFailed:
		return true
	}
	return false
}

// TestWork acts as an end-to-end verification of the entire process:
// adding a job, picking up the job, processing the job, and closing the client
func TestWork_Integration(t *testing.T) {
	// Set up the logger since we're using the real client
	client.SetLogger(logger)

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
	jobArgs := worker_types.JobEnqueueArgs{ID: id, ACOID: cmsID, BBBasePath: uuid.New()}

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

	// Save and set environment variables using conf.SetEnv and defer to reset them
	defer func(archiveDir, stagingDir, payloadDir string) {
		conf.SetEnv(t, "FHIR_ARCHIVE_DIR", archiveDir)
		conf.SetEnv(t, "FHIR_STAGING_DIR", stagingDir)
		conf.SetEnv(t, "FHIR_PAYLOAD_DIR", payloadDir)
	}(conf.GetEnv("FHIR_ARCHIVE_DIR"), conf.GetEnv("FHIR_STAGING_DIR"), conf.GetEnv("FHIR_PAYLOAD_DIR"))

	conf.SetEnv(t, "FHIR_ARCHIVE_DIR", archivePath)
	conf.SetEnv(t, "FHIR_STAGING_DIR", stagingPath)
	conf.SetEnv(t, "FHIR_PAYLOAD_DIR", payloadPath)

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
	mockJob := &river.Job[worker_types.CleanupJobArgs]{
		Args: worker_types.CleanupJobArgs{},
	}

	// Call the Work function
	err := cleanupJobWorker.Work(context.Background(), mockJob)

	// Assert that there was no error
	assert.NoError(t, err)

	// Assert that all expectations were met
	mockCleanupJob.AssertExpectations(t)
	mockArchiveExpiring.AssertExpectations(t)
}

func TestGetCutOffTime(t *testing.T) {
	// Save and set environment variable using conf.SetEnv and defer to reset it
	defer func(origValue string) {
		conf.SetEnv(t, "ARCHIVE_THRESHOLD_HR", origValue)
	}(conf.GetEnv("ARCHIVE_THRESHOLD_HR"))

	// Test with default value
	conf.SetEnv(t, "ARCHIVE_THRESHOLD_HR", "")
	expectedCutoff := time.Now().Add(-24 * time.Hour)
	actualCutoff := getCutOffTime()
	assert.WithinDuration(t, expectedCutoff, actualCutoff, time.Second, "Cutoff time should be 24 hours ago by default")

	// Test with custom value
	conf.SetEnv(t, "ARCHIVE_THRESHOLD_HR", "48")
	expectedCutoff = time.Now().Add(-48 * time.Hour)
	actualCutoff = getCutOffTime()
	assert.WithinDuration(t, expectedCutoff, actualCutoff, time.Second, "Cutoff time should be 48 hours ago")
}

func TestGetAWSParams(t *testing.T) {
	defer func(env, workflowAlerts, localStackEndpoint string) {
		conf.SetEnv(t, "ENV", env)
		conf.SetEnv(t, "workflow-alerts", workflowAlerts)
		os.Setenv("LOCAL_STACK_ENDPOINT", localStackEndpoint)
	}(conf.GetEnv("ENV"), conf.GetEnv("workflow-alerts"), os.Getenv("LOCAL_STACK_ENDPOINT"))

	t.Run("Local Environment", func(t *testing.T) {
		conf.SetEnv(t, "ENV", "local")
		expectedToken := "local-token"
		conf.SetEnv(t, "workflow-alerts", expectedToken)

		token, err := getAWSParams()
		assert.NoError(t, err)
		assert.Equal(t, expectedToken, token)
	})
}

func TestNewCleanupJobWorker(t *testing.T) {
	worker := NewCleanupJobWorker()

	assert.NotNil(t, worker)
	assert.NotNil(t, worker.cleanupJob)
	assert.NotNil(t, worker.archiveExpiring)
}
