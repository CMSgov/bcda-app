package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/conf"
)

var destBucket = "bcda-aco-credentials"
var kmsAliasName = "alias/bcda-aco-creds-kms"
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

	return awsParams{slackToken, ssasURL, clientID, clientSecret, ssasPEM}, nil
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

func getKMSID(service kmsiface.KMSAPI) (string, error) {
	kmsInput := &kms.ListAliasesInput{}
	kmsResult, err := service.ListAliases(kmsInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kms.ErrCodeDependencyTimeoutException:
				log.Error(kms.ErrCodeDependencyTimeoutException, aerr.Error())
			case kms.ErrCodeInvalidMarkerException:
				log.Error(kms.ErrCodeInvalidMarkerException, aerr.Error())
			case kms.ErrCodeInternalException:
				log.Error(kms.ErrCodeInternalException, aerr.Error())
			case kms.ErrCodeInvalidArnException:
				log.Error(kms.ErrCodeInvalidArnException, aerr.Error())
			case kms.ErrCodeNotFoundException:
				log.Error(kms.ErrCodeNotFoundException, aerr.Error())
			default:
				log.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			log.Error(err.Error())
		}
		return "", err
	}

	var id string
	for _, alias := range kmsResult.Aliases {
		if *alias.AliasName == kmsAliasName {
			id = *alias.TargetKeyId
			break
		}
	}

	return id, nil
}

func putObject(service s3iface.S3API, creds string, kmsID string) (string, error) {
	bucketSuffix := adjustedEnv()

	s3Input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(creds)),
		Bucket: aws.String(fmt.Sprintf("%s/%s", destBucket, bucketSuffix)),
		Key:    aws.String(kmsID),
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
