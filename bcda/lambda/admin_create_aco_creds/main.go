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

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

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

	err = setupEnvironment(params)
	if err != nil {
		log.Errorf("Unable to setupEnvironment properly: %+v", err)
		return "", err
	}

	s3Service := s3.New(session)
	slackClient := slack.New(params.slackToken)

	s3Path, err := handleCreateACOCreds(ctx, data, s3Service, slackClient, params.credsBucket)
	if err != nil {
		log.Errorf("Failed to handle Create ACO creds: %+v", err)
		return "", err
	}

	log.Info("Completed Create ACO Creds administrative task")

	return fmt.Sprintf("Client credentials for %s can be found at: %s", data.ACOID, s3Path), nil
}

func handleCreateACOCreds(
	ctx context.Context,
	data payload,
	s3Service s3iface.S3API,
	notifier Notifier,
	credsBucket string,
) (string, error) {
	_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started Create ACO Creds lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier start message: %+v", err)
	}

	creds, err := auth.GetProvider().FindAndCreateACOCredentials(data.ACOID, data.IPs)
	if err != nil {
		log.Errorf("Error creating ACO creds: %+v", err)

		_, _, slackErr := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: Create ACO Creds lambda in %s env.", os.Getenv("ENV")), false),
		)
		if slackErr != nil {
			log.Errorf("Error sending notifier failure message: %+v", slackErr)
		}

		return "", err
	}

	s3Path, err := putObject(s3Service, data.ACOID, creds, credsBucket)
	if err != nil {
		log.Errorf("Error putting object: %+v", err)

		_, _, slackErr := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: Create ACO Creds lambda in %s env.", os.Getenv("ENV")), false),
		)
		if slackErr != nil {
			log.Errorf("Error sending notifier failure message: %+v", slackErr)
		}

		return "", err
	}

	_, _, err = notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Success: Create ACO Creds lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier success message: %+v", err)
	}

	return s3Path, nil
}
