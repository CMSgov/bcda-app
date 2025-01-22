package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
	"github.com/slack-go/slack"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"

	log "github.com/sirupsen/logrus"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

type payload struct {
	DenyACOIDs []string `json:"deny_aco_ids"`
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
	log.Info("Starting ACO Deny administrative task")

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

	conn, err := pgx.Connect(ctx, params.DBURL)
	if err != nil {
		log.Errorf("Unable to connect to database: %+v", err)
		return err
	}
	defer conn.Close(ctx)

	slackClient := slack.New(params.SlackToken)

	err = handleACODenies(ctx, conn, data, slackClient)
	if err != nil {
		log.Errorf("Failed to handle ACO denies: %+v", err)
		return err
	}

	log.Info("Completed ACO Deny administrative task")

	return nil
}

func handleACODenies(ctx context.Context, conn PgxConnection, data payload, notifier Notifier) error {
	_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started ACO Deny lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier start message: %+v", err)
	}

	err = denyACOs(ctx, conn, data)
	if err != nil {
		log.Errorf("Error finding and denying ACOs: %+v", err)

		_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
			fmt.Sprintf("Failed: ACO Deny List lambda in %s env.", os.Getenv("ENV")), false),
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
