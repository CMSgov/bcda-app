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
	slackUtils "github.com/CMSgov/bcda-app/bcda/slack"

	log "github.com/sirupsen/logrus"
)

type payload struct {
	Name    string  `json:"name"`
	CMSID   string  `json:"cms_id"`
	CleanUp *string `json:"clean_up,omitempty"`
}

type awsParams struct {
	dbURL      string
	slackToken string
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

	if data.CleanUp == nil {

		// run the regular logic (non-rollback transaction)
		err = handleCreateACO(ctx, conn, data, id)
		if err != nil {
			slackUtils.SendSlackMessage(slackClient, slackUtils.OperationsChannel, fmt.Sprintf("%s: Create ACO lambda in %s env.", slackUtils.FailureMsg, os.Getenv("ENV")), false)
			log.Errorf("Failed to handle Create ACO: %+v", err)
			return err
		}
		slackUtils.SendSlackMessage(slackClient, slackUtils.OperationsChannel, fmt.Sprintf("%s: Create ACO lambda in %s env.", slackUtils.SuccessMsg, os.Getenv("ENV")), true)

	} else {
		// create a rollbackable transaction
		tx, cErr := conn.Begin(ctx)
		if cErr != nil {
			log.Errorf("Failed to create transaction: %v+", cErr)
			return err
		}

		err = handleCreateACO(ctx, tx, data, id)
		if err != nil {
			log.Errorf("Failed to handle Create ACO: %+v", err)
			return err
		}

		err := tx.Rollback(ctx)
		if err != nil {
			log.Errorf("Failed to rollback transaction: %v+", err)
			return err
		}
	}

	log.Info("Completed Create ACO administrative task")

	return nil
}

func handleCreateACO(ctx context.Context, conn PgxConnection, data payload, id uuid.UUID) error {

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

	err := createACO(context.Background(), conn, aco)
	if err != nil {
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
