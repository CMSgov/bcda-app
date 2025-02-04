package main

import (
	"fmt"
	"strings"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
)

var destBucket = "bcda-aco-credentials"
var kmsAliasName = "alias/bcda-aco-creds-kms"

func getAWSParams(session *session.Session) (awsParams, error) {
	env := conf.GetEnv("ENV")

	if env == "local" {
		return awsParams{}, nil
	}

	slackToken, err := bcdaaws.GetParameter(session, "/slack/token/workflow-alerts")
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{slackToken}, nil
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
	bucketSuffix := conf.GetEnv("ENV")
	if bucketSuffix == "sbx" {
		bucketSuffix = "opensbx"
	}

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
