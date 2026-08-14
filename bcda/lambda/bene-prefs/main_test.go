package main

import (
	"context"
	"testing"
	"time"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleOptOutImport(t *testing.T) {
	assert := assert.New(t)
	s3Client := &bcdaaws.MockS3Client{}
	repo := &models.MockRepository{}
	path := "../../../shared_files/synthetic1800MedicareFiles/test2/"

	env := uuid.NewString()
	cleanupEnv := testUtils.SetEnvVars(t, []testUtils.EnvVar{{Name: "ENV", Value: env}})
	defer cleanupEnv()

	res, err := handleOptOutImport(context.Background(), repo, s3Client, path)
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
