package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	msgr "github.com/CMSgov/bcda-app/bcda/slackmessenger"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type rotationSystem struct {
	SystemId   string `json:"system_id"`
	CredsParam string `json:"creds_param"`
}

type shortCreds struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"` // #nosec G117
}

type RotateSSASCredsHandler struct {
	logger      *logrus.Entry
	ssmClient   bcdaaws.CustomSSMClient
	ssas        client.SSASHTTPClient
	slackClient *slack.Client
}

func main() {
	ctx := context.Background()
	handler, err := initHandler(ctx)
	if err != nil {
		logrus.Fatalf("failed to initialize handler: %v", err)
	}
	lambda.Start(handler.Handle)
}

func initHandler(ctx context.Context) (*RotateSSASCredsHandler, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Errorf("failed to load default config: %+v", err)
		return nil, err
	}
	ssmClient := ssm.NewFromConfig(cfg)
	params, err := getAWSParams(ctx, ssmClient)
	if err != nil {
		logger.Errorf("failed to retrieve AWS params: %+v", err)
		return nil, err
	}

	err = setupEnv(params)
	if err != nil {
		logger.Errorf("unable to setupEnvironment properly: %+v", err)
		return nil, err
	}

	slackClient := slack.New(params.slackToken)
	if err != nil {
		logger.Errorf("failed to create slack client: %+v", err)
		return nil, err
	}
	ssas, err := client.NewSSASClient()
	if err != nil {
		logger.Errorf("failed to create SSAS client: %s", err)
		return nil, err
	}

	handler := RotateSSASCredsHandler{logger: logger, ssmClient: ssmClient, ssas: ssas, slackClient: slackClient}
	return &handler, nil
}

func (h RotateSSASCredsHandler) Handle(ctx context.Context) error {
	successes, failures := 0, 0
	rotationSystems, err := getRotationSystemsParam(ctx, h.ssmClient)
	if err != nil {
		h.logger.Errorf("failed to retrieve AWS params: %+v", err)
		failureMsg := "failed to retrieve AWS params in rotate-ssas-creds lambda"
		msgr.SendFailureToAlerts(h.slackClient, failureMsg)
		return err
	}

	for _, rs := range rotationSystems {
		err := h.rotateCreds(ctx, rs)
		if err != nil {
			h.logger.Errorf("failed to rotate creds for system %s", rs.CredsParam)
			failures += 1
		} else {
			h.logger.Infof("successfully rotated creds for system %s", rs.CredsParam)
			successes += 1
		}
	}

	reportMsg := fmt.Sprintf("ssas creds rotation complete. successes: %d, failures: %d", successes, failures)
	h.logger.Info(reportMsg)
	if failures > 0 {
		msgr.SendFailureToAlerts(h.slackClient, reportMsg)
	} else {
		msgr.SendSuccessToOperations(h.slackClient, reportMsg)
	}

	return nil
}

func (h RotateSSASCredsHandler) rotateCreds(ctx context.Context, rs rotationSystem) error {
	if len(rs.SystemId) == 0 {
		h.logger.Errorf("failed to get system id for system %s", rs.CredsParam)
	}
	if len(rs.CredsParam) == 0 {
		h.logger.Errorf("failed to get creds param for a system")
	}
	var newCreds shortCreds
	credsBytes, err := h.ssas.ResetCredentials(rs.SystemId)
	if err != nil {
		h.logger.Errorf("failed to reset creds for system %s: %+v", rs.CredsParam, err)
		return err
	}

	err = json.Unmarshal(credsBytes, &newCreds)
	if err != nil {
		h.logger.Errorf("failed to unmarshal new creds for system %s", rs.CredsParam)
		return err
	}
	newValueBytes, err := json.Marshal(newCreds)
	if err != nil {
		h.logger.Errorf("failed to re-marshal new creds for system %s", rs.CredsParam)
		return err
	}
	newValue := string(newValueBytes)
	return updateCredsParam(ctx, h.ssmClient, rs.CredsParam, newValue)
}

func configureLogger(env, appName string) *logrus.Entry {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})

	logger.SetReportCaller(true)

	return logger.WithFields(logrus.Fields{
		"application": appName,
		"environment": env,
	})
}
