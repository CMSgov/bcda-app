package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/conf"
)

var pemFilePath = "/tmp/BCDA_CA_FILE.pem"

func getAWSParams(session *session.Session) (awsParams, error) {
	env := adjustedEnv()

	if env == "local" {
		return awsParams{}, nil
	}

	slackToken, err := bcdaaws.GetParameter(session, "/slack/token/workflow-alerts")
	if err != nil {
		return awsParams{}, err
	}

	ssasURL, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/SSAS_URL", env))
	if err != nil {
		return awsParams{}, err
	}

	clientID, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_CLIENT_ID", env))
	if err != nil {
		return awsParams{}, err
	}

	clientSecret, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_SECRET", env))
	if err != nil {
		return awsParams{}, err
	}

	ssasPEM, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/api/BCDA_CA_FILE.pem", env))
	if err != nil {
		return awsParams{}, err
	}

	credsBucket, err := bcdaaws.GetParameter(session, fmt.Sprintf("/bcda/%s/aco_creds_bucket", env))
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{slackToken, ssasURL, clientID, clientSecret, ssasPEM, credsBucket}, nil
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

func putObject(service s3iface.S3API, acoID string, creds string, credsBucket string) (string, error) {
	s3Input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(creds)),
		Bucket: aws.String(credsBucket),
		Key:    aws.String(fmt.Sprintf("%s-creds", acoID)),
	}

	result, err := service.PutObject(s3Input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			log.Error(err.Error())
		}
		return "", err
	}

	return result.String(), nil
}

func adjustedEnv() string {
	env := conf.GetEnv("ENV")
	if env == "sbx" {
		env = "opensbx"
	}
	return env
}
