package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/slack-go/slack"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/bcdacli"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/s3"
)

var slackChannel = "C034CFU945C" // #bcda-alerts
var destBucket = "bcda-aco-credentials"
var kmsAliasName = "alias/bcda-aco-creds-kms"

// var awsRegion = "us-east-1"

type payload struct {
	ACOID string   `json:"aco_id"`
	IPs   []string `josn:"ips"`
}

type awsParams struct {
	SlackToken string
}

type Notifier interface {
	PostMessageContext(context.Context, string, ...slack.MsgOption) (string, string, error)
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event json.RawMessage) (string, error) {
	log.SetFormatter(&log.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	log.Info("Starting Create ACO Creds administrative task")

	var data payload
	err := json.Unmarshal(event, &data)
	if err != nil {
		log.Errorf("Failed to unmarshal event: %v", err)
		return "", err
	}

	session, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return "", err
	}

	params, err := getAWSParams(session)
	if err != nil {
		log.Errorf("Unable to extract slack token from parameter store: %+v", err)
		return "", err
	}

	kmsService := kms.New(session)
	s3Service := s3.New(session)
	slackClient := slack.New(params.SlackToken)

	s3Path, err := handleCreateACOCreds(ctx, data, kmsService, s3Service, slackClient)
	if err != nil {
		log.Errorf("Failed to handle ACO denies: %+v", err)
		return "", err
	}

	log.Info("Completed Create ACO Creds administrative task")

	return fmt.Sprintf("Client credentials for %s can be found at: %s", data.ACOID, s3Path), nil
}

func handleCreateACOCreds(ctx context.Context, data payload, kmsService *kms.KMS, s3Service *s3.S3, notifier Notifier) (string, error) {
	_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started Create ACO Creds lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier start message: %+v", err)
	}

	creds, err := bcdacli.GenerateClientCredentials(data.ACOID, data.IPs)
	if err != nil {
		log.Errorf("Error creating ACO creds: %+v", err)

		_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: Create ACO Creds List lambda in %s env.", os.Getenv("ENV")), false),
		)
		if err != nil {
			log.Errorf("Error sending notifier failure message: %+v", err)
		}

		return "", err
	}

	kmsInput := &kms.ListAliasesInput{}
	kmsResult, err := kmsService.ListAliases(kmsInput)
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

	var kmsID string
	for _, alias := range kmsResult.Aliases {
		if *alias.AliasName == kmsAliasName {
			kmsID = *alias.TargetKeyId
			break
		}
	}

	s3Input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(creds)),
		Bucket: aws.String(destBucket),
		Key:    aws.String(kmsID),
	}
	s3Result, err := s3Service.PutObject(s3Input)
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

	_, _, err = notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Success: Create ACO Creds List lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier success message: %+v", err)
	}

	return s3Result.String(), nil
}

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
