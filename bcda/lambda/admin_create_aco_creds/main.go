package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/slack-go/slack"

	"github.com/CMSgov/bcda-app/bcda/auth"
	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/database"
	msgr "github.com/CMSgov/bcda-app/bcda/slackmessenger"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type payload struct {
	ACOID string   `json:"aco_id"`
	IPs   []string `json:"ips"`
}

type awsParams struct {
	slackToken   string
	ssasURL      string
	clientID     string
	clientSecret string
	ssasPEM      string
	credsBucket  string
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

	params, err := getAWSParams(ctx)
	if err != nil {
		log.Errorf("Unable to extract slack token from parameter store: %+v", err)
		return "", err
	}

	err = setupEnvironment(params)
	if err != nil {
		log.Errorf("Unable to setupEnvironment properly: %+v", err)
		return "", err
	}

	provider := auth.NewProvider(database.Connect())

	cfg, err := bcdaaws.NewAWSConfig(ctx, "", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		log.Errorf("Unable to setupEnvironment properly: %+v", err)
		return "", err
	}

	s3Service := s3.NewFromConfig(cfg)
	slackClient := slack.New(params.slackToken)

	s3Path, err := handleCreateACOCreds(ctx, data, provider, s3Service, params.credsBucket)
	if err != nil {
		msgr.SendSlackMessage(slackClient, msgr.OperationsChannel, fmt.Sprintf("%s: Create ACO Credentials lambda in %s env.", msgr.FailureMsg, os.Getenv("ENV")), msgr.Danger)
		log.Errorf("Failed to handle Create ACO creds: %+v", err)
		return "", err
	}

	msgr.SendSlackMessage(slackClient, msgr.OperationsChannel, fmt.Sprintf("%s: Create ACO Credentials lambda in %s env.", msgr.SuccessMsg, os.Getenv("ENV")), msgr.Good)

	log.Info("Completed Create ACO Creds administrative task")

	return fmt.Sprintf("Client credentials for %s can be found at: %s", data.ACOID, s3Path), nil
}

func handleCreateACOCreds(
	ctx context.Context,
	data payload,
	provider auth.Provider,
	s3Service s3iface.S3API,
	credsBucket string,
) (string, error) {

	creds, err := provider.FindAndCreateACOCredentials(data.ACOID, data.IPs)
	if err != nil {
		log.Errorf("Error creating ACO creds: %+v", err)

		return "", err
	}

	s3Path, err := putObject(s3Service, data.ACOID, creds, credsBucket)
	if err != nil {
		log.Errorf("Error putting object: %+v", err)

		return "", err
	}

	return s3Path, nil
}
