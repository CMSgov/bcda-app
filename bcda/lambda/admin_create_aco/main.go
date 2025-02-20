package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
	"github.com/pborman/uuid"
	"github.com/slack-go/slack"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"

	log "github.com/sirupsen/logrus"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

type payload struct {
	Name  string `json:"name"`
	CMSID string `json:"cms_id"`
}

type awsParams struct {
	dbURL      string
	slackToken string
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

	conn, err := pgx.Connect(ctx, params.dbURL)
	if err != nil {
		log.Errorf("Unable to connect to database: %+v", err)
		return err
	}
	defer conn.Close(ctx)

	slackClient := slack.New(params.slackToken)
	id := uuid.NewRandom()

	err = handleCreateACO(ctx, conn, data, id, slackClient)
	if err != nil {
		log.Errorf("Failed to handle Create ACO: %+v", err)
		return err
	}

	log.Info("Completed Create ACO administrative task")

	return nil
}

func handleCreateACO(ctx context.Context, conn PgxConnection, data payload, id uuid.UUID, notifier Notifier) error {
	_, _, err := notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Started Create ACO lambda in %s env.", os.Getenv("ENV")), false),
	)
	if err != nil {
		log.Errorf("Error sending notifier start message: %+v", err)
	}

	if data.Name == "" {
		return errors.New("ACO name must be provided")
	}

	if data.CMSID == "" {
		return errors.New("CMSID must be provided")
	}

	var cmsIDPt *string
	if data.CMSID != "" {
		match := service.IsSupportedACO(data.CMSID)
		if !match {
			return errors.New("ACO CMS ID is invalid")
		}
		cmsIDPt = &data.CMSID
	}

	aco := models.ACO{Name: data.Name, CMSID: cmsIDPt, UUID: id, ClientID: id.String()}

	err = createACO(context.Background(), conn, aco)
	if err != nil {
		return err
	}

	_, _, err = notifier.PostMessageContext(ctx, slackChannel, slack.MsgOptionText(
		fmt.Sprintf("Success: Create ACO lambda in %s env.", os.Getenv("ENV")), false),
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
