package queueing

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCleanupJob struct {
	mock.Mock
}

type MockArchiveExpiring struct {
	mock.Mock
}

func (m *MockCleanupJob) CleanupJob(db *sql.DB, maxDate time.Time, currentStatus, newStatus models.JobStatus, rootDirsToClean ...string) error {
	args := m.Called(db, maxDate, currentStatus, newStatus, rootDirsToClean)
	return args.Error(0)
}

func (m *MockArchiveExpiring) ArchiveExpiring(db *sql.DB, maxDate time.Time) error {
	args := m.Called(db, maxDate)
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

	mockCleanupJob.On("CleanupJob", mock.Anything, mock.AnythingOfType("time.Time"), models.JobStatusArchived, models.JobStatusExpired, []string{archivePath, stagingPath}).Return(nil)
	mockCleanupJob.On("CleanupJob", mock.Anything, mock.AnythingOfType("time.Time"), models.JobStatusFailed, models.JobStatusFailedExpired, []string{stagingPath, payloadPath}).Return(nil)
	mockCleanupJob.On("CleanupJob", mock.Anything, mock.AnythingOfType("time.Time"), models.JobStatusCancelled, models.JobStatusCancelledExpired, []string{stagingPath, payloadPath}).Return(nil)
	mockArchiveExpiring.On("ArchiveExpiring", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil)

	// Create a worker instance
	cleanupJobWorker := &CleanupJobWorker{
		cleanupJob:      mockCleanupJob.CleanupJob,
		archiveExpiring: mockArchiveExpiring.ArchiveExpiring,
		db:              database.Connect(),
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

func TestNewCleanupJobWorker(t *testing.T) {
	worker := NewCleanupJobWorker(database.Connect())

	assert.NotNil(t, worker)
	assert.NotNil(t, worker.cleanupJob)
	assert.NotNil(t, worker.archiveExpiring)
}
