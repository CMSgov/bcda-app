package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	bp "github.com/CMSgov/bcda-app/bcda/bene-prefs"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type BenePrefsImportHandler struct {
	db       *sql.DB
	logger   *logrus.Entry
	repo     models.Repository
	s3Client bcdaaws.CustomS3Client
}

func main() {
	ctx := context.Background()
	handler, err := initHandler(ctx)
	if err != nil {
		logrus.Fatalf("failed to initialize handler: %v", err)
	}
	if handler.db != nil {
		defer handler.db.Close()
	}

	lambda.Start(handler.importHandler)
}

func initHandler(ctx context.Context) (*BenePrefsImportHandler, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("Failed to load Default Config")
		return nil, err
	}
	ssmClient := ssm.NewFromConfig(cfg)

	s3AssumeRoleArn, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/bcda/%s/bene-prefs/sensitive/iam_bucket_role_arn", env))
	if err != nil {
		logger.Errorf("error getting param: %+v", err)
		return nil, err
	}
	stsClient := sts.NewFromConfig(cfg)
	appCreds := stscreds.NewAssumeRoleProvider(stsClient, s3AssumeRoleArn)

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Credentials = appCreds
	})

	dbURL, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/bcda/%s/sensitive/api/DATABASE_URL", env))
	if err != nil {
		logger.Error("failed to load DB URL")
		return nil, err
	}

	err = os.Setenv("DATABASE_URL", dbURL)
	if err != nil {
		logger.Errorf("error setting dbURL env var: %+v", err)
		return nil, err
	}

	db := database.Connect()
	repo := postgres.NewRepository(db)

	return &BenePrefsImportHandler{
		db:       db,
		logger:   logger,
		repo:     repo,
		s3Client: s3Client,
	}, nil
}

func (h *BenePrefsImportHandler) importHandler(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
	if len(sqsEvent.Records) == 0 {
		h.logger.Info("no SQS records found, skipping safely")
		return "", nil
	}

	event, err := bcdaaws.ParseSQSEvent(sqsEvent)
	if err != nil {
		h.logger.Errorf("failed to parse S3 event: %v", err)
		return "", err
	} else if event == nil {
		h.logger.Info("no S3 event found, skipping safely")
		return "", nil
	}

	for _, e := range event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			dir := bcdaaws.ParseS3Directory(e.S3.Bucket.Name, e.S3.Object.Key)
			h.logger.Infof("Reading %s event for directory %s", e.EventName, dir)
			return h.importDir(ctx, dir)
		}
	}

	h.logger.Info("No ObjectCreated events found, skipping safely.")
	return "", nil
}

func (h *BenePrefsImportHandler) importDir(ctx context.Context, s3ImportPath string) (string, error) {
	importer := bp.BenePrefsImporter{
		FileClient:           h.s3Client,
		Repo:                 h.repo,
		Logger:               h.logger,
		ImportStatusInterval: utils.GetEnvInt("SUPPRESS_IMPORT_STATUS_RECORDS_INTERVAL", 1000),
	}

	s, f, sk, err := importer.ImportDirectory(ctx, s3ImportPath)
	if err != nil {
		errMsg := fmt.Errorf("error importing directory %s: %w", s3ImportPath, err)
		h.logger.Error(errMsg)
		return errMsg.Error(), err
	}

	resultMsg := fmt.Sprintf("Completed Bene-Prefs suppression data import.  Files imported: %v, Files failed: %v, Files skipped: %v", s, f, sk)
	h.logger.Info(resultMsg)
	return resultMsg, nil
}

func configureLogger(env, appName string) *logrus.Entry {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})

	log.SetReportCaller(true)

	return log.WithFields(logrus.Fields{
		"application": appName,
		"environment": env,
	})
}
