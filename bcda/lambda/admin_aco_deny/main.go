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
	slackUtils "github.com/CMSgov/bcda-app/bcda/slack"

	log "github.com/sirupsen/logrus"
)

type payload struct {
	DenyACOIDs []string `json:"deny_aco_ids"`
}

type awsParams struct {
	DBURL      string
	SlackToken string
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

	err = handleACODenies(ctx, conn, data)
	if err != nil {
		slackUtils.SendSlackMessage(slackClient, slackUtils.OperationsChannel, fmt.Sprintf("%s: Deny ACO lambda in %s env.", slackUtils.FailureMsg, os.Getenv("ENV")), false)

		log.Errorf("Failed to handle ACO denies: %+v", err)
		return err
	}

	slackUtils.SendSlackMessage(slackClient, slackUtils.OperationsChannel, fmt.Sprintf("%s: Deny ACO lambda in %s env.", slackUtils.SuccessMsg, os.Getenv("ENV")), true)

	log.Info("Completed ACO Deny administrative task")

	return nil
}

func handleACODenies(ctx context.Context, conn PgxConnection, data payload) error {

	err := denyACOs(ctx, conn, data)
	if err != nil {
		log.Errorf("Error finding and denying ACOs: %+v", err)

		return err
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
