package main

import (
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

type mockS3 struct {
	s3iface.S3API
}

func (m *mockS3) PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func TestPutObject(t *testing.T) {
	mock := &mockS3{}

	result, err := putObject(mock, "test-filename", "test-creds", "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, result, "{\n\n}")
}

func TestAdjustedEnv(t *testing.T) {
	origEnv := conf.GetEnv("ENV")

	conf.SetEnv(t, "ENV", "dev")
	resultEnv := adjustedEnv()
	assert.Equal(t, resultEnv, "dev")

	conf.SetEnv(t, "ENV", "test")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "test")

	conf.SetEnv(t, "ENV", "sbx")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "opensbx")

	conf.SetEnv(t, "ENV", "prod")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "prod")

	conf.SetEnv(t, "ENV", origEnv)
}

func TestSetupEnvironment(t *testing.T) {
	// store env vars to restore later
	origSSASURL := os.Getenv("SSAS_URL")
	origBCDASSASClientID := os.Getenv("BCDA_SSAS_CLIENT_ID")
	origBCDASSASSecret := os.Getenv("BCDA_SSAS_SECRET")
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	origBCDACAFile := os.Getenv("BCDA_CA_FILE")

	err := setupEnvironment(awsParams{
		ssasURL:      "test-SSAS_URL",
		clientID:     "test-BCDA_SSAS_CLIENT_ID",
		clientSecret: "test-BCDA_SSAS_SECRET",
	})
	assert.Nil(t, err)

	assert.Equal(t, "test-SSAS_URL", os.Getenv("SSAS_URL"))
	assert.Equal(t, "test-BCDA_SSAS_CLIENT_ID", os.Getenv("BCDA_SSAS_CLIENT_ID"))
	assert.Equal(t, "test-BCDA_SSAS_SECRET", os.Getenv("BCDA_SSAS_SECRET"))
	assert.Equal(t, "true", os.Getenv("SSAS_USE_TLS"))
	assert.Equal(t, pemFilePath, os.Getenv("BCDA_CA_FILE"))

	assert.FileExists(t, pemFilePath)

	// restore original env vars
	err = os.Setenv("SSAS_URL", origSSASURL)
	assert.Nil(t, err)
	err = os.Setenv("BCDA_SSAS_CLIENT_ID", origBCDASSASClientID)
	assert.Nil(t, err)
	err = os.Setenv("BCDA_SSAS_SECRET", origBCDASSASSecret)
	assert.Nil(t, err)
	err = os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	assert.Nil(t, err)
	err = os.Setenv("BCDA_CA_FILE", origBCDACAFile)
	assert.Nil(t, err)
}
