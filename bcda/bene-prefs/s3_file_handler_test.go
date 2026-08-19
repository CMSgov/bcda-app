package beneprefs

import (
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// unfortunately due to the aws s3 mock being used a lot of these tests are pretty limited in coverage
func mockHandler() *S3FileHandler {
	return &S3FileHandler{
		Client: &bcdaaws.MockS3Client{},
		Logger: logrus.New(),
	}
}

func TestLoggerFunctions(t *testing.T) {
	logger, hook := test.NewNullLogger()
	handler := &S3FileHandler{
		Logger: logger,
	}

	handler.Infof("test info message")
	assert.Equal(t, "test info message", hook.LastEntry().Message)

	handler.Warningf("test warning message")
	assert.Equal(t, "test warning message", hook.LastEntry().Message)

	handler.Errorf("test error message")
	assert.Equal(t, "test error message", hook.LastEntry().Message)
}

func TestLoadBenePrefsFiles(t *testing.T) {
	handler := mockHandler()

	path := "s3://test-bucket/test-prefix/"
	suppressList, skipped, err := handler.LoadBenePrefsFiles(t.Context(), path)
	assert.NoError(t, err)
	assert.NotNil(t, suppressList)
	assert.Equal(t, 0, skipped)
}

func TestListFiles(t *testing.T) {
	handler := mockHandler()

	bucket := "test-bucket"
	prefix := "test-prefix/"
	s3Objects, err := handler.ListFiles(t.Context(), bucket, prefix)
	assert.NoError(t, err)
	assert.Len(t, s3Objects, 0)
}

func TestOpenFile(t *testing.T) {
	handler := mockHandler()

	fileBytes, f, err := handler.OpenFile(t.Context(), &models.BenePrefsFilenameMetadata{})
	assert.ErrorContains(t, err, "is empty or does not exist")
	assert.Nil(t, fileBytes)
	assert.Nil(t, f)
}

func TestOpenFileBytes(t *testing.T) {
	handler := mockHandler()
	path := "s3://test-bucket/test-prefix/test-file.txt"

	fileBytes, err := handler.OpenFileBytes(t.Context(), path)
	assert.ErrorContains(t, err, "is empty or does not exist")
	assert.Len(t, fileBytes, 0)
}

func TestCleanupBenePrefsFiles(t *testing.T) {
	handler := mockHandler()

	err := handler.CleanupBenePrefsFiles(t.Context(), []*models.BenePrefsFilenameMetadata{})
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	handler := mockHandler()
	t.Setenv("S3_DELETE_TIMEOUT", "1")

	err := handler.Delete(t.Context(), "s3://test-bucket/test-prefix/test-file.txt")
	assert.ErrorContains(t, err, "exceeded max wait time for ObjectNotExists waiter")
}
