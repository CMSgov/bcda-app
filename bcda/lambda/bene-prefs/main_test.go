package main

import (
	"context"
	"testing"
	"time"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportHandler(t *testing.T) {
	assert := assert.New(t)
	s3Client := &bcdaaws.MockS3Client{}
	repo := &models.MockRepository{}

	env := uuid.NewString()
	cleanupEnv := testUtils.SetEnvVars(t, []testUtils.EnvVar{{Name: "ENV", Value: env}})
	defer cleanupEnv()

	h := &BenePrefsImportHandler{
		logger:   logrus.NewEntry(logrus.New()),
		repo:     repo,
		s3Client: s3Client,
	}

	// test with empty event, should return no error and message indicating no ObjectCreated events found
	res, err := h.importHandler(context.Background(), events.SQSEvent{})
	assert.Nil(err)
	assert.Equal("", res)

	// test with actual path, should return no error and ~successful import
	event := testUtils.GetSQSEvent(t, "test-bucket", "shared_files/synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	res, err = h.importHandler(context.Background(), event)
	assert.Nil(err)
	assert.Contains(res, "Completed Bene-Prefs suppression data import")
	// due to using mock aws s3 we dont have any actual files to import, so the counts will be 0
	assert.Contains(res, "Files imported: 0")
	assert.Contains(res, "Files failed: 0")
	assert.Contains(res, "Files skipped: 0")
}

func TestImportDir(t *testing.T) {
	assert := assert.New(t)
	s3Client := &bcdaaws.MockS3Client{}
	repo := &models.MockRepository{}
	path := "../../../shared_files/synthetic1800MedicareFiles/test2/"

	env := uuid.NewString()
	cleanupEnv := testUtils.SetEnvVars(t, []testUtils.EnvVar{{Name: "ENV", Value: env}})
	defer cleanupEnv()

	h := &BenePrefsImportHandler{
		logger:   logrus.NewEntry(logrus.New()),
		repo:     repo,
		s3Client: s3Client,
	}
	res, err := h.importDir(context.Background(), path)
	assert.Nil(err)
	assert.Contains(res, constants.CompleteMedSupDataImp)
	// due to using mock aws s3 we dont have any actual files to import, so the counts will be 0
	assert.Contains(res, "Files imported: 0")
	assert.Contains(res, "Files failed: 0")
	assert.Contains(res, "Files skipped: 0")
}

func TestConfigureLogger(t *testing.T) {
	logger := configureLogger("test_env", "test_app_name")
	require.NotNil(t, logger)
	assert.Equal(t, "test_env", logger.Data["environment"])
	assert.Equal(t, "test_app_name", logger.Data["application"])
	formatter, ok := logger.Logger.Formatter.(*logrus.JSONFormatter)
	require.True(t, ok, "expected *logrus.JSONFormatter")
	assert.True(t, formatter.DisableHTMLEscape)
	assert.Equal(t, time.RFC3339Nano, formatter.TimestampFormat)
	assert.True(t, logger.Logger.ReportCaller)
}
