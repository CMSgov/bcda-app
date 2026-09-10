package main

import (
	"context"
	"encoding/json"
	"errors"
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
	SystemId  string `json:"system_id"`
	CredsName string `json:"creds_name"`
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
	env         string
	keyAlias    string
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
	env, err := getRequiredEnv("ENV")
	if err != nil {
		logrus.Fatalf("failed to fetch environment variable: %+v", err)
	}

	appName, err := getRequiredEnv("APP_NAME")
	if err != nil {
		logrus.Fatalf("failed to fetch environment variable: %+v", err)
	}

	keyAlias, err := getRequiredEnv("KEY_ALIAS")
	if err != nil {
		logrus.Fatalf("failed to fetch environment variable: %+v", err)
	}

	logger := configureLogger(env, appName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Errorf("failed to load default config: %+v", err)
		return nil, err
	}
	ssmClient := ssm.NewFromConfig(cfg)
	params, err := getAWSParams(ctx, ssmClient, env)
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
	if slackClient == nil {
		logger.Errorf("failed to create slack client: %+v", err)
		return nil, err
	}
	ssas, err := client.NewSSASClient()
	if err != nil {
		logger.Errorf("failed to create SSAS client: %s", err)
		return nil, err
	}

	handler := RotateSSASCredsHandler{logger: logger, ssmClient: ssmClient, ssas: ssas, slackClient: slackClient, env: env, keyAlias: keyAlias}
	return &handler, nil
}

func (h RotateSSASCredsHandler) Handle(ctx context.Context) error {
	successes, failures := 0, 0
	rotationSystems, err := getRotationSystemsParam(ctx, h.ssmClient, h.env)
	if err != nil {
		h.logger.Errorf("failed to retrieve AWS params: %+v", err)
		failureMsg := "failed to retrieve AWS params in rotate-ssas-creds lambda"
		msgr.SendFailureToAlerts(h.slackClient, failureMsg)
		return err
	}

	for _, rs := range rotationSystems {
		err := h.rotateCreds(ctx, rs)
		if err != nil {
			h.logger.Errorf("failed to rotate creds for system %s", rs.CredsName)
			failures += 1
		} else {
			h.logger.Infof("successfully rotated creds for system %s", rs.CredsName)
			successes += 1
		}
	}

	reportMsg := fmt.Sprintf("ssas creds rotation complete. successes: %d, failures: %d", successes, failures)
	h.logger.Info(reportMsg)
	if failures > 0 {
		msgr.SendFailureToAlerts(h.slackClient, reportMsg)
		return errors.New("failed to rotate one or more ssas creds -- see lambda logs for details")
	} else {
		msgr.SendSuccessToOperations(h.slackClient, reportMsg)
	}

	return nil
}

func (h RotateSSASCredsHandler) rotateCreds(ctx context.Context, rs rotationSystem) error {
	if len(rs.SystemId) == 0 {
		return fmt.Errorf("failed to get system id for system %s", rs.CredsName)
	}
	if len(rs.CredsName) == 0 {
		return fmt.Errorf("failed to get creds param for a system")
	}
	var newCreds shortCreds
	credsBytes, err := h.ssas.ResetCredentials(rs.SystemId)
	if err != nil {
		h.logger.Errorf("failed to reset creds for system %s: %+v", rs.CredsName, err)
		return err
	}

	err = json.Unmarshal(credsBytes, &newCreds)
	if err != nil {
		h.logger.Errorf("failed to unmarshal new creds for system %s", rs.CredsName)
		return err
	}
	newValueBytes, err := json.Marshal(newCreds)
	if err != nil {
		h.logger.Errorf("failed to re-marshal new creds for system %s", rs.CredsName)
		return err
	}
	newValue := string(newValueBytes)
	return updateCredsParam(ctx, h.ssmClient, h.env, h.keyAlias, rs.CredsName, newValue)
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

func getRequiredEnv(envVar string) (string, error) {
	value := conf.GetEnv(envVar)
	if len(value) == 0 {
		return "", fmt.Errorf("failed to get %s from environment", envVar)
	}
	return value, nil
}
