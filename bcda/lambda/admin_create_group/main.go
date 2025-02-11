package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/pborman/uuid"

	log "github.com/sirupsen/logrus"
)

var slackChannel = "C034CFU945C" // #bcda-alerts

type payload struct {
	GroupID   string `json:"group_id"`
	GroupName string `json:"group_name"`
	ACO_ID    string `json:"aco_id"` // CMS_ID or UUID
}

type awsParams struct {
	DBURL        string
	SlackToken   string
	ssasURL      string
	clientID     string
	clientSecret string
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
	log.Info("Starting Create Group administrative task")

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

	err = os.Setenv("SSAS_URL", params.ssasURL)
	if err != nil {
		log.Errorf("Error setting SSAS URL env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_SSAS_CLIENT_ID", params.clientID)
	if err != nil {
		log.Errorf("Error setting SSAS URL env var: %+v", err)
		return err
	}
	err = os.Setenv("BCDA_SSAS_SECRET", params.clientSecret)
	if err != nil {
		log.Errorf("Error setting SSAS URL env var: %+v", err)
		return err
	}

	conn, err := pgx.Connect(ctx, params.DBURL)
	if err != nil {
		log.Errorf("Unable to connect to database: %+v", err)
		return err
	}
	defer conn.Close(ctx)

	slackClient := slack.New(params.SlackToken)
	db := database.Connection
	r := postgres.NewRepository(db)
	ssas, err := client.NewSSASClient()

	if err != nil {
		log.Errorf("failed to create SSAS client: %s", err)
	}

	sendSlackMessage(slackClient, fmt.Sprintf("Started Create Group lambda in %s env.", os.Getenv("ENV")))
	err = handleCreateGroup(ssas, r, data)
	if err != nil {
		sendSlackMessage(slackClient, fmt.Sprintf("Failed: Create Group lambda in %s env.", os.Getenv("ENV")))
		log.Errorf("Failed to Create Group: %+v", err)
		return err
	}

	sendSlackMessage(slackClient, fmt.Sprintf("Success: Create Group lambda in %s env.", os.Getenv("ENV")))
	log.Info("Completed Create Group administrative task")

	return nil
}

func handleCreateGroup(c client.SSASHTTPClient, r *postgres.Repository, data payload) error {
	var (
		aco *models.ACO
		err error
	)

	if data.GroupID == "" || data.GroupName == "" || data.ACO_ID == "" {
		return errors.New("missing one or more required field(s): group_id, group_name, aco_id")
	}

	if match := service.IsSupportedACO(data.ACO_ID); match {
		aco, err = r.GetACOByCMSID(context.Background(), data.ACO_ID)
		if err != nil {
			return err
		}
	} else if match, err := regexp.MatchString("[0-9a-f]{6}-([0-9a-f]{4}-){3}[0-9a-f]{12}", data.ACO_ID); err == nil && match {
		aco, err = r.GetACOByUUID(context.Background(), uuid.Parse(data.ACO_ID))
		if err != nil {
			return err
		}
	} else {
		return errors.New("ACO ID is invalid or not found")
	}

	b, err := c.CreateGroup(data.GroupID, data.GroupName, *aco.CMSID)
	if err != nil {
		return err
	}

	var g map[string]interface{}
	err = json.Unmarshal(b, &g)
	if err != nil {
		return err
	}

	if val, ok := g["group_id"]; ok {
		ssasID := val.(string)
		if aco.UUID != nil {
			aco.GroupID = ssasID

			err := r.UpdateACO(context.Background(), aco.UUID,
				map[string]interface{}{"group_id": ssasID})
			if err != nil {
				return errors.Wrapf(err, "group %s was created, but ACO could not be updated", ssasID)
			}
		}
	} else {
		log.Errorf("failed to get group_id: %s", err)
	}

	return nil
}

func getAWSParams() (awsParams, error) {
	env := conf.GetEnv("ENV")

	if env == "local" {
		return awsParams{conf.GetEnv("DATABASE_URL"), "", "", "", ""}, nil
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

	ssasURL, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/bcda/%s/api/SSAS_URL", env))
	if err != nil {
		return awsParams{}, err
	}

	clientID, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_CLIENT_ID", env))
	if err != nil {
		return awsParams{}, err
	}

	clientSecret, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_SECRET", env))
	if err != nil {
		return awsParams{}, err
	}

	return awsParams{dbURL, slackToken, ssasURL, clientID, clientSecret}, nil
}

func sendSlackMessage(sc *slack.Client, msg string) {
	_, _, err := sc.PostMessageContext(context.Background(), slackChannel, slack.MsgOptionText(fmt.Sprint(msg), false))
	if err != nil {
		log.Errorf("Failed to send slack message: %+v", err)
	}
}
