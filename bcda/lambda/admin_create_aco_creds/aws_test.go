package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestPutObject(t *testing.T) {
	client := testUtils.TestS3Client(t, testUtils.TestAWSConfig(t))

	bucketInput := &s3.CreateBucketInput{
		Bucket: aws.String("test-bucket"),
	}
	_, err := client.CreateBucket(t.Context(), bucketInput)
	assert.Nil(t, err)

	result, err := putObject(t.Context(), client, "test-filename", "test-creds", "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, result, "test-bucket/test-filename-creds")
}

func TestGetAWSParams(t *testing.T) {
	env := conf.GetEnv("ENV")

	cleanupParam1 := testUtils.SetParameter(t, "/slack/token/workflow-alerts", "slack-val")
	t.Cleanup(func() { cleanupParam1() })
	cleanupParam2 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/aco_creds_bucket", env), "test-CREDS_BUCKET")
	t.Cleanup(func() { cleanupParam2() })
	cleanupParam3 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/SSAS_URL", env), "test-SSAS_URL")
	t.Cleanup(func() { cleanupParam3() })
	cleanupParam4 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_CLIENT_ID", env), "test-BCDA_SSAS_CLIENT_ID")
	t.Cleanup(func() { cleanupParam4() })
	cleanupParam5 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_SECRET", env), "test-BCDA_SSAS_SECRET")
	t.Cleanup(func() { cleanupParam5() })
	cleanupParam6 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/BCDA_CA_FILE.pem", env), "test-BCDA_CA_FILE")
	t.Cleanup(func() { cleanupParam6() })
	cleanupParam7 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env), "test-DB_URL")
	t.Cleanup(func() { cleanupParam7() })

	params, err := getAWSParams(context.Background())
	assert.Nil(t, err)

	assert.Equal(t, "slack-val", params.slackToken)
	assert.Equal(t, "test-CREDS_BUCKET", params.credsBucket)
	assert.Equal(t, "test-SSAS_URL", params.ssasURL)
	assert.Equal(t, "test-BCDA_SSAS_CLIENT_ID", params.clientID)
	assert.Equal(t, "test-BCDA_SSAS_SECRET", params.clientSecret)
	assert.Equal(t, "test-BCDA_CA_FILE", params.ssasPEM)
	assert.Equal(t, "test-DB_URL", params.dbURL)
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

	err := setupEnvironment(awsParams{
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
