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
	msgr "github.com/CMSgov/bcda-app/bcda/slackmessenger"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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

	params, err := getAWSParams(ctx)
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
		msgr.SendSlackMessage(slackClient, msgr.OperationsChannel, fmt.Sprintf("%s: Deny ACO lambda in %s env.", msgr.FailureMsg, os.Getenv("ENV")), msgr.Danger)

		log.Errorf("Failed to handle ACO denies: %+v", err)
		return err
	}

	msgr.SendSlackMessage(slackClient, msgr.OperationsChannel, fmt.Sprintf("%s: Deny ACO lambda in %s env.", msgr.SuccessMsg, os.Getenv("ENV")), msgr.Good)

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

func getAWSParams(ctx context.Context) (awsParams, error) {
	env := conf.GetEnv("ENV")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return awsParams{}, err
	}
	ssmClient := ssm.NewFromConfig(cfg)

	dbURLName := fmt.Sprintf("/bcda/%s/sensitive/api/DATABASE_URL", env)
	slackParamName := "/slack/token/workflow-alerts"
	paramNames := []string{slackParamName, dbURLName}
	params, err := bcdaaws.GetParameters(ctx, ssmClient, paramNames)
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{params[dbURLName], params[slackParamName]}, nil
}
