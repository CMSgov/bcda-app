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
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/suppression"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	lambda.Start(optOutImportHandler)
}

func optOutImportHandler(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)

	s3Event, err := bcdaaws.ParseSQSEvent(sqsEvent)
	if err != nil {
		logger.Errorf("Failed to parse S3 event: %v", err)
		return "", err
	} else if s3Event == nil {
		logger.Info("No S3 event found, skipping safely.")
		return "", nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("Failed to load Default Config")
		return "", err
	}
	ssmClient := ssm.NewFromConfig(cfg)

	s3AssumeRoleArn, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/cclf-import/bcda/%s/bfd-bucket-role-arn", env))
	if err != nil {
		logger.Errorf("error getting param: %+v", err)
		return "", err
	}
	stsClient := sts.NewFromConfig(cfg)
	appCreds := stscreds.NewAssumeRoleProvider(stsClient, s3AssumeRoleArn)

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Credentials = appCreds
	})

	dbURL, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/bcda/%s/sensitive/api/DATABASE_URL", env))
	if err != nil {
		logger.Error("failed to load DB URL")
		return "", err
	}

	err = os.Setenv("DATABASE_URL", dbURL)
	if err != nil {
		logger.Errorf("error setting dbURL env var: %+v", err)
		return "", err
	}

	db := database.Connect()

	for _, e := range s3Event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			dir := bcdaaws.ParseS3Directory(e.S3.Bucket.Name, e.S3.Object.Key)
			logger.Infof("Reading %s event for directory %s", e.EventName, dir)
			return handleOptOutImport(ctx, db, s3Client, dir)
		}
	}

	logger.Info("No ObjectCreated events found, skipping safely.")
	return "", nil
}

func handleOptOutImport(ctx context.Context, db *sql.DB, s3Client *s3.Client, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	repo := postgres.NewRepository(db)

	importer := suppression.OptOutImporter{
		FileHandler: &optout.S3FileHandler{
			Client: s3Client,
			Logger: logger,
		},
		Saver: &suppression.BCDASaver{
			Repo: repo,
		},
		Logger:               logger,
		ImportStatusInterval: utils.GetEnvInt("SUPPRESS_IMPORT_STATUS_RECORDS_INTERVAL", 1000),
	}

	s, f, sk, err := importer.ImportSuppressionDirectory(ctx, s3ImportPath)
	result := fmt.Sprintf("Completed 1-800-MEDICARE suppression data import.\nFiles imported: %v\nFiles failed: %v\nFiles skipped: %v\n", s, f, sk)
	logger.Info(result)
	return result, err
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
