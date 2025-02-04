package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/slack-go/slack"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/bcdacli"

	log "github.com/sirupsen/logrus"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

type payload struct {
	Name  string `json:"name"`
	CMSID string `json:"cms_id"`
}

type awsParams struct {
	DBURL      string
	SlackToken string
}

type Notifier interface {
	PostMessageContext(context.Context, string, ...slack.MsgOption) (string, string, error)
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event json.RawMessage) error {
	log.SetFormatter(&log.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	log.Info("Starting Create ACO Task")

	var data payload
	err := json.Unmarshal(event, &data)
	if err != nil {
		log.Errorf("Failed to unmarshal event: %v", err)
		return err
	}

	params, err := getAWSParams()
	if err != nil {
		log.Errorf("Unable to extract DB URL from parameter store: %+v", err)
		return err
	}

	slackClient := slack.New(params.SlackToken)

	err = handleCreateACO(ctx, data, slackClient)
	if err != nil {
		log.Errorf("Failed to handle Create ACO: %+v", err)
		return err
	}

	log.Info("Completed Create ACO administrative task")

	return nil
}

func handleCreateACO(ctx context.Context, data payload, notifier Notifier) error {
	_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started Create ACO lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier start message: %+v", err)
	}

	// TODO: acoUuid potentially needs to be returned
	_, err = bcdacli.CreateACO(data.Name, data.CMSID)
	if err != nil {
		log.Errorf("Error creating ACO: %+v", err)

		_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: Create ACO lambda in %s env.", os.Getenv("ENV")), false),
		)
		if err != nil {
			log.Errorf("Error sending notifier failure message: %+v", err)
		}

		return err
	}

	_, _, err = notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Success: ACO Deny List lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier success message: %+v", err)
	}

	return nil
}

func getAWSParams() (awsParams, error) {
	env := conf.GetEnv("ENV")

	if env == "local" {
		return awsParams{conf.GetEnv("DATABASE_URL"), ""}, nil
	}

	bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return awsParams{}, err
	}

	dbURL, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env))
	if err != nil {
		return awsParams{}, err
	}

	slackToken, err := bcdaaws.GetParameter(bcdaSession, "/slack/token/workflow-alerts")
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{dbURL, slackToken}, nil
}
