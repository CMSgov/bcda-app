package main

import (
	"context"
	"os"
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/stretchr/testify/assert"
)

func TestPutObject(t *testing.T) {
	client := &bcdaaws.MockS3Client{}

	result, err := putObject(t.Context(), client, "test-filename", "test-creds", "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, result, "test-bucket/test-filename-creds")
}

func TestGetAWSParams(t *testing.T) {
	ssmClient := bcdaaws.MockSSMClient{Params: map[string]string{
		"/slack/token/workflow-alerts":                  "test-slack-token",
		"/bcda/local/sensitive/api/DATABASE_URL":        "test-db-url",
		"/bcda/local/sensitive/api/SSAS_URL":            "test-ssas-url",
		"/bcda/local/sensitive/api/BCDA_SSAS_CLIENT_ID": "test-client-id",
		"/bcda/local/sensitive/api/BCDA_SSAS_SECRET":    "test-client-secret",
		"/bcda/local/sensitive/api/BCDA_CA_FILE.pem":    "test-ca-file",
		"/bcda/local/sensitive/aco_creds_bucket":        "test-creds-bucket",
	}} // #nosec G101

	params, err := getAWSParams(context.Background(), &ssmClient)
	assert.Nil(t, err)

	assert.Equal(t, "test-slack-token", params.slackToken)
	assert.Equal(t, "test-db-url", params.dbURL)
	assert.Equal(t, "test-ssas-url", params.ssasURL)
	assert.Equal(t, "test-client-id", params.clientID)
	assert.Equal(t, "test-client-secret", params.clientSecret)
	assert.Equal(t, "test-ca-file", params.ssasPEM)
	assert.Equal(t, "test-creds-bucket", params.credsBucket)

}

func TestAdjustedEnv(t *testing.T) {
	origEnv := conf.GetEnv("ENV")
	t.Cleanup(func() {
		conf.SetEnv(t, "ENV", origEnv)
	})

	conf.SetEnv(t, "ENV", "dev")
	resultEnv := adjustedEnv()
	assert.Equal(t, resultEnv, "dev")

	conf.SetEnv(t, "ENV", "test")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "test")

	conf.SetEnv(t, "ENV", "sbx")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "sandbox")

	conf.SetEnv(t, "ENV", "prod")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "prod")

	conf.SetEnv(t, "ENV", "asdf")
	resultEnv = adjustedEnv()
	assert.Equal(t, resultEnv, "asdf")
}

func TestSetupEnvironment(t *testing.T) {
	// store env vars to restore later
	origSSASURL := os.Getenv("SSAS_URL")
	origDBURL := os.Getenv("DATABASE_URL")
	origBCDASSASClientID := os.Getenv("BCDA_SSAS_CLIENT_ID")
	origBCDASSASSecret := os.Getenv("BCDA_SSAS_SECRET")
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	origBCDACAFile := os.Getenv("BCDA_CA_FILE")

	t.Cleanup(func() {
		// restore original env vars
		err := os.Setenv("SSAS_URL", origSSASURL)
		assert.Nil(t, err)
		err = os.Setenv("DATABASE_URL", origDBURL)
		assert.Nil(t, err)
		err = os.Setenv("BCDA_SSAS_CLIENT_ID", origBCDASSASClientID)
		assert.Nil(t, err)
		err = os.Setenv("BCDA_SSAS_SECRET", origBCDASSASSecret)
		assert.Nil(t, err)
		err = os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
		assert.Nil(t, err)
		err = os.Setenv("BCDA_CA_FILE", origBCDACAFile)
		assert.Nil(t, err)
	})

	err := setupEnvironment(awsParams{ // #nosec G101
		ssasURL:      "test-SSAS_URL",
		dbURL:        "test-DB_URL",
		clientID:     "test-BCDA_SSAS_CLIENT_ID",
		clientSecret: "test-BCDA_SSAS_SECRET",
	})
	assert.Nil(t, err)

	assert.Equal(t, "test-SSAS_URL", os.Getenv("SSAS_URL"))
	assert.Equal(t, "test-DB_URL", os.Getenv("DATABASE_URL"))
	assert.Equal(t, "test-BCDA_SSAS_CLIENT_ID", os.Getenv("BCDA_SSAS_CLIENT_ID"))
	assert.Equal(t, "test-BCDA_SSAS_SECRET", os.Getenv("BCDA_SSAS_SECRET"))
	assert.Equal(t, "true", os.Getenv("SSAS_USE_TLS"))
	assert.Equal(t, pemFilePath, os.Getenv("BCDA_CA_FILE"))

	assert.FileExists(t, pemFilePath)
}
