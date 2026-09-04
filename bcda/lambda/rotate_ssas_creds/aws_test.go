package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/stretchr/testify/assert"
)

func TestGetAWSParams(t *testing.T) {
	ssmClient := bcdaaws.MockSSMClient{Params: map[string]string{
		"/bcda/local/sensitive/api/SSAS_URL":            "test-ssas-url",
		"/bcda/local/sensitive/api/BCDA_SSAS_CLIENT_ID": "test-client-id",
		"/bcda/local/sensitive/api/BCDA_SSAS_SECRET":    "test-client-secret",
		"/bcda/local/sensitive/api/BCDA_CA_FILE.pem":    "test-ca-file",
		"/slack/token/workflow-alerts":                  "test-slack-token",
	}} // #nosec G101

	params, err := getAWSParams(context.Background(), &ssmClient)
	assert.Nil(t, err)

	assert.Equal(t, "test-ssas-url", params.ssasURL)
	assert.Equal(t, "test-client-id", params.clientID)
	assert.Equal(t, "test-client-secret", params.clientSecret)
	assert.Equal(t, "test-ca-file", params.ssasPEM)
	assert.Equal(t, "test-slack-token", params.slackToken)
}

func TestSetupEnv(t *testing.T) {
	// store env vars to restore later
	origSSASURL := os.Getenv("SSAS_URL")
	origBCDASSASClientID := os.Getenv("BCDA_SSAS_CLIENT_ID")
	origBCDASSASSecret := os.Getenv("BCDA_SSAS_SECRET")
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	origBCDACAFile := os.Getenv("BCDA_CA_FILE")

	t.Cleanup(func() {
		// restore original env vars
		err := os.Setenv("SSAS_URL", origSSASURL)
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

	err := setupEnv(awsParams{ // #nosec G101
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
}

func TestGetRotationSystemsParam(t *testing.T) {
	stored := rotationSystem{SystemId: "11", CredsParam: "TestRotationACO"}
	jsonBytes, _ := json.Marshal([]rotationSystem{stored})
	ssmClient := bcdaaws.MockSSMClient{Params: map[string]string{
		"/bcda/local/sensitive/rotation_systems": string(jsonBytes),
	}}
	rotationSystems, err := getRotationSystemsParam(t.Context(), &ssmClient)
	assert.Nil(t, err)
	assert.Equal(t, stored.SystemId, rotationSystems[0].SystemId)
	assert.Equal(t, stored.CredsParam, rotationSystems[0].CredsParam)
}

func TestUpdateCredsParam(t *testing.T) {
	name := "TestACOCreds"
	value := "test-creds-value"
	ssmClient := bcdaaws.MockSSMClient{}
	err := updateCredsParam(t.Context(), &ssmClient, name, value)
	assert.Nil(t, err)

	fullParamName := "/bcda/local/creds/TestACOCreds"
	input := ssm.GetParameterInput{Name: &fullParamName}
	param, err := ssmClient.GetParameter(t.Context(), &input)
	assert.Nil(t, err)
	assert.Equal(t, value, *param.Parameter.Value)

}
