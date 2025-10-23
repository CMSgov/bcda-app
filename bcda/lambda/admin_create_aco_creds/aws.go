package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	log "github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/conf"
)

var pemFilePath = "/tmp/BCDA_CA_FILE.pem"

func getAWSParams(ctx context.Context) (awsParams, error) {
	env := adjustedEnv()
	if env == "local" {
		return awsParams{}, nil
	}

	slackTokenName := "/slack/token/workflow-alerts"
	ssasURLName := fmt.Sprintf("/bcda/%s/api/SSAS_URL", env)
	clientIDName := fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_CLIENT_ID", env)
	clientSecretName := fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_SECRET", env)
	ssasPEMName := fmt.Sprintf("/bcda/%s/api/BCDA_CA_FILE.pem", env)
	credsBucketName := fmt.Sprintf("/bcda/%s/aco_creds_bucket", env)

	paramNames := []string{
		slackTokenName,
		ssasURLName,
		clientIDName,
		clientSecretName,
		ssasPEMName,
		credsBucketName,
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return awsParams{}, err
	}
	ssmClient := ssm.NewFromConfig(cfg)

	params, err := bcdaaws.GetParameters(ctx, ssmClient, paramNames)
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{
		params[slackTokenName],
		params[ssasURLName],
		params[clientIDName],
		params[clientSecretName],
		params[ssasPEMName],
		params[credsBucketName],
	}, nil
	// output, err := client.GetParameters(ctx, input)
	// if err != nil {
	// 	return awsParams{}, err
	// }
	// slackToken := getParamFromOutput(output, "/slack/token/workflow-alerts")
	// ssasURL := getParamFromOutput(output, "/bcda/%s/api/SSAS_URL")
	// clientID := getParamFromOutput(output, "/bcda/%s/api/BCDA_SSAS_CLIENT_ID")
	// clientSecret := getParamFromOutput(output, "/bcda/%s/api/BCDA_SSAS_SECRET")
	// ssasPEM := getParamFromOutput(output, "/bcda/%s/api/BCDA_CA_FILE.pem")
	// credsBucket := getParamFromOutput(output, "/bcda/%s/aco_creds_bucket")

	// return awsParams{slackToken, ssasURL, clientID, clientSecret, ssasPEM, credsBucket}, nil

	// slackToken, err := bcdaaws.GetParameter(session, "/slack/token/workflow-alerts")
	// if err != nil {
	// 	return awsParams{}, err
	// }

	// ssasURL, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/SSAS_URL", env))
	// if err != nil {
	// 	return awsParams{}, err
	// }

	// clientID, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_CLIENT_ID", env))
	// if err != nil {
	// 	return awsParams{}, err
	// }

	// clientSecret, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_SECRET", env))
	// if err != nil {
	// 	return awsParams{}, err
	// }

	// ssasPEM, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/BCDA_CA_FILE.pem", env))
	// if err != nil {
	// 	return awsParams{}, err
	// }

	// credsBucket, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/aco_creds_bucket", env))
	// if err != nil {
	// 	return awsParams{}, err
	// }

	// return awsParams{slackToken, ssasURL, clientID, clientSecret, ssasPEM, credsBucket}, nil
}

func setupEnvironment(params awsParams) error {
	// need to set these env vars for the initialization of SSASClient and for its requests to SSAS
	err := os.Setenv("SSAS_URL", params.ssasURL)
	if err != nil {
		log.Errorf("Error setting SSAS_URL env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_SSAS_CLIENT_ID", params.clientID)
	if err != nil {
		log.Errorf("Error setting BCDA_SSAS_CLIENT_ID env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_SSAS_SECRET", params.clientSecret)
	if err != nil {
		log.Errorf("Error setting BCDA_SSAS_SECRET env var: %+v", err)
		return err
	}
	err = os.Setenv("SSAS_USE_TLS", "true")
	if err != nil {
		log.Errorf("Error setting SSAS_USE_TLS env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_CA_FILE", pemFilePath)
	if err != nil {
		log.Errorf("Error setting SSAS_USE_TLS env var: %+v", err)
	}

	// parameter store returns the value of the paremeter and SSAS expects a file, so we need to create it
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

func putObject(ctx context.Context, client *s3.Client, acoID, creds, credsBucket string) (string, error) {
	s3Input := &s3.PutObjectInput{
		Body:   strings.NewReader(creds),
		Bucket: aws.String(credsBucket),
		Key:    aws.String(fmt.Sprintf("%s-creds", acoID)),
	}

	_, err := client.PutObject(ctx, s3Input)
	if err != nil {
		return "", err
	}

	return (credsBucket + "/" + acoID + "-creds"), nil
}

func adjustedEnv() string {
	env := conf.GetEnv("ENV")
	if env == "sbx" {
		env = "opensbx"
	}
	return env
}
