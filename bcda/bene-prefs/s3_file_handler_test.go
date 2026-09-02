package beneprefs

import (
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// unfortunately due to the aws s3 mock being used a lot of these tests are pretty limited in coverage
func mockHelper() bcdaaws.S3Helper {
	return bcdaaws.S3Helper{
		Client: &bcdaaws.MockS3Client{},
		Logger: logrus.New(),
	}
}

func TestLoadBenePrefsFiles(t *testing.T) {
	helper := mockHelper()

	path := "s3://test-bucket/test-prefix/"
	suppressList, skipped, err := LoadBenePrefsFiles(t.Context(), helper, path)
	assert.NoError(t, err)
	assert.NotNil(t, suppressList)
	assert.Equal(t, 0, skipped)
}

func TestCleanupBenePrefsFiles(t *testing.T) {
	helper := mockHelper()

	err := CleanupBenePrefsFiles(t.Context(), helper, []*models.BenePrefsFilenameMetadata{})
	assert.NoError(t, err)
}
