package main

import (
	"context"
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
)

func main() {
	if os.Getenv("LOCAL_STACK_ENDPOINT") != "" {
		res, err := handleOptOutImport(os.Getenv("BFD_BUCKET_ROLE_ARN"), os.Getenv("BFD_S3_IMPORT_PATH"))
		if err != nil {
			fmt.Printf("Failed to run opt out import: %s\n", err.Error())
		} else {
			fmt.Println(res)
		}
	} else {
		lambda.Start(optOutImportHandler)
	}
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

	for _, e := range s3Event.Records {
		if e.EventName == "ObjectCreated:Put" {
			s3AssumeRoleArn, err := loadBfdS3Params()
			if err != nil {
				return "", err
			}

			parts := strings.Split(e.S3.Object.Key, "/")

			if len(parts) == 1 {
				return handleOptOutImport(s3AssumeRoleArn, e.S3.Bucket.Name)
			} else {
				directory := fmt.Sprintf("%s/%s", e.S3.Bucket.Name, parts[0])
				return handleOptOutImport(s3AssumeRoleArn, directory)
			}
		}
	}

	logger.Info("No ObjectCreated:Put events found, skipping safely.")
	return "", nil
}

func loadBfdS3Params() (string, error) {
	env := conf.GetEnv("ENV")

	bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return "", err
	}

	param, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/opt-out-import/bcda/%s/bfd-bucket-role-arn", env))
	if err != nil {
		return "", err
	}

	return param, nil
}

func handleOptOutImport(s3AssumeRoleArn, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	repo := postgres.NewRepository(database.Connection)

	importer := suppression.OptOutImporter{
		FileHandler: &optout.S3FileHandler{
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

	s, f, sk, err := importer.ImportSuppressionDirectory(s3ImportPath)
	logger.Infof("Completed 1-800-MEDICARE suppression data import.\nFiles imported: %v\nFiles failed: %v\nFiles skipped: %v\n", s, f, sk)
	return "success", err
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