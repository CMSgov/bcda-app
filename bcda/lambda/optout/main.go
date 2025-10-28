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
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func main() {
	lambda.Start(optOutImportHandler)
}

func optOutImportHandler(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	db := database.Connect()

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
	s3Client := s3.NewFromConfig(cfg)

	for _, e := range s3Event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			dir := bcdaaws.ParseS3Directory(e.S3.Bucket.Name, e.S3.Object.Key)
			logger.Infof("Reading %s event for directory %s", e.EventName, dir)
			return handleOptOutImport(ctx, db, s3Client, ssmClient, dir)
		}
	}

	logger.Info("No ObjectCreated events found, skipping safely.")
	return "", nil
}

func handleOptOutImport(ctx context.Context, db *sql.DB, s3Client *s3.Client, ssmClient *ssm.Client, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	repo := postgres.NewRepository(db)

	s3AssumeRoleArn, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/opt-out-import/bcda/%s/bfd-bucket-role-arn", env))
	if err != nil {
		logger.Errorf("error getting param: %+v", err)
		return "", err
	}

	importer := suppression.OptOutImporter{
		FileHandler: &optout.S3FileHandler{
			Client:        s3Client,
			Logger:        logger,
			Endpoint:      os.Getenv("LOCAL_STACK_ENDPOINT"),
			AssumeRoleArn: s3AssumeRoleArn,
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
