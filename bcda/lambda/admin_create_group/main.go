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

	slackToken, err := getAWSParams()
	if err != nil {
		log.Errorf("Failed to retrieve parameter: %+v", err)
		return err
	}

	slackClient := slack.New(slackToken)
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

func getAWSParams() (string, error) {
	env := conf.GetEnv("ENV")

	bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return "", err
	}

	err = os.Setenv("SSAS_USE_TLS", "true")
	if err != nil {
		log.Errorf("Error setting SSAS_USE_TLS env var: %+v", err)
		return "", err
	}

	envVars := []string{"SSAS_URL", "BCDA_SSAS_CLIENT_ID", "BCDA_SSAS_SECRET", "BCDA_CA_FILE", "BCDA_CA_FILE.pem"}
	for _, v := range envVars {
		envVar, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/bcda/%s/api/%s", env, v))
		if err != nil {
			return "", err
		}
		err = os.Setenv(v, envVar)
		if err != nil {
			log.Errorf("Error setting %s env var: %+v", envVar, err)
			return "", err
		}

	}

	ca := conf.GetEnv("BCDA_CA_FILE.pem")
	log.Infof("cert info: %s ", ca)

	slackToken, err := bcdaaws.GetParameter(bcdaSession, "/slack/token/workflow-alerts")
	if err != nil {
		return "", err
	}

	return slackToken, nil
}

func sendSlackMessage(sc *slack.Client, msg string) {
	// _, _, err := sc.PostMessageContext(context.Background(), slackChannel, slack.MsgOptionText(fmt.Sprint(msg), false))
	// if err != nil {
	// 	log.Errorf("Failed to send slack message: %+v", err)
	// }
}
