package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	log "github.com/sirupsen/logrus"
)

var pemFilePath = "/tmp/BCDA_CA_FILE.pem"

type awsParams struct {
	ssasURL      string
	clientID     string
	clientSecret string
	ssasPEM      string
	slackToken   string
}

func getAWSParams(ctx context.Context, client bcdaaws.CustomSSMClient) (awsParams, error) {
	env := conf.GetEnv("ENV")

	ssasURLName := fmt.Sprintf("/bcda/%s/sensitive/api/SSAS_URL", env)
	clientIDName := fmt.Sprintf("/bcda/%s/sensitive/api/BCDA_SSAS_CLIENT_ID", env)
	clientSecretName := fmt.Sprintf("/bcda/%s/sensitive/api/BCDA_SSAS_SECRET", env)
	ssasPEMName := fmt.Sprintf("/bcda/%s/sensitive/api/BCDA_CA_FILE.pem", env)
	slackParamName := "/slack/token/workflow-alerts"

	paramNames := []string{
		ssasURLName,
		clientIDName,
		clientSecretName,
		ssasPEMName,
		slackParamName,
	}

	params, err := bcdaaws.GetParameters(ctx, client, paramNames)
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{
		ssasURL:      params[ssasURLName],
		clientID:     params[clientIDName],
		clientSecret: params[clientSecretName],
		ssasPEM:      params[ssasPEMName],
		slackToken:   params[slackParamName],
	}, nil
}

func setupEnv(params awsParams) error {
	err := os.Setenv("SSAS_URL", params.ssasURL)
	if err != nil {
		log.Errorf("error setting SSAS_URL env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_SSAS_CLIENT_ID", params.clientID)
	if err != nil {
		log.Errorf("error setting BCDA_SSAS_CLIENT_ID env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_SSAS_SECRET", params.clientSecret)
	if err != nil {
		log.Errorf("error setting BCDA_SSAS_SECRET env var: %+v", err)
		return err
	}
	err = os.Setenv("SSAS_USE_TLS", "true")
	if err != nil {
		log.Errorf("error setting SSAS_USE_TLS env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_CA_FILE", pemFilePath)
	if err != nil {
		log.Errorf("error setting BCDA_CA_FILE env var: %+v", err)
	}

	// parameter store returns the value of the parameter and SSAS expects a file, so we need to create it
	// nosec in use because lambda creates a tmp dir already
	f, err := os.Create(pemFilePath) // #nosec
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(params.ssasPEM))
	if err != nil {
		return err
	}
	return nil
}

func getRotationSystemsParam(ctx context.Context, ssmClient bcdaaws.CustomSSMClient) ([]rotationSystem, error) {
	env := conf.GetEnv("ENV")
	rotationSystemsName := fmt.Sprintf("/bcda/%s/sensitive/rotation_systems", env)
	param, err := bcdaaws.GetParameter(ctx, ssmClient, rotationSystemsName)
	if err != nil {
		log.Errorf("failed to get rotation systems parameter: %+v", err)
		return nil, err
	}

	var rotationSystems []rotationSystem
	err = json.Unmarshal([]byte(param), &rotationSystems)
	if err != nil {
		log.Errorf("failed to unmarshal cred systems: %+v", err)
		return nil, err
	}
	return rotationSystems, nil
}

func updateCredsParam(ctx context.Context, ssmClient bcdaaws.CustomSSMClient, name string, value string) error {
	fullCredsParam := fmt.Sprintf("/bcda/%s/creds/%s", conf.GetEnv("ENV"), name)
	input := &ssm.PutParameterInput{
		Name:        aws.String(fullCredsParam),
		Value:       aws.String(value),
		Type:        types.ParameterTypeSecureString,
		Overwrite:   aws.Bool(true),
		Description: aws.String(fmt.Sprintf("Creds for system %s", name)),
	}

	_, err := ssmClient.PutParameter(ctx, input)
	if err != nil {
		log.Errorf("failed to update creds param for system %s", name)
		return err
	}
	return nil
}
