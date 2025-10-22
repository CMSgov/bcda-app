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
	// Localstack is a local-development server that mimics AWS. The endpoint variable
	// should only be set in local development to avoid making external calls to a real AWS account.
	if os.Getenv("LOCAL_STACK_ENDPOINT") != "" {
		res, err := handleOptOutImport(context.Background(), database.Connect(), os.Getenv("BFD_BUCKET_ROLE_ARN"), os.Getenv("BFD_S3_IMPORT_PATH"))
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
	db := database.Connect()

	s3Event, err := bcdaaws.ParseSQSEvent(sqsEvent)

	if err != nil {
		logger.Errorf("Failed to parse S3 event: %v", err)
		return "", err
	} else if s3Event == nil {
		logger.Info("No S3 event found, skipping safely.")
		return "", nil
	}

	for _, e := range s3Event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			s3AssumeRoleArn, err := loadBfdS3Params(ctx)
			if err != nil {
				return "", err
			}

			dir := bcdaaws.ParseS3Directory(e.S3.Bucket.Name, e.S3.Object.Key)
			logger.Infof("Reading %s event for directory %s", e.EventName, dir)
			return handleOptOutImport(ctx, db, s3AssumeRoleArn, dir)
		}
	}

	logger.Info("No ObjectCreated events found, skipping safely.")
	return "", nil
}

func loadBfdS3Params(ctx context.Context) (string, error) {
	env := conf.GetEnv("ENV")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}
	ssmClient := ssm.NewFromConfig(cfg)

	return bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/cclf-import/bcda/%s/bfd-bucket-role-arn", env))
}

func handleOptOutImport(ctx context.Context, db *sql.DB, s3AssumeRoleArn, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	repo := postgres.NewRepository(db)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("error loading default config: ", err)
		return "", err
	}
	client := s3.NewFromConfig(cfg)

	importer := suppression.OptOutImporter{
		FileHandler: &optout.S3FileHandler{
			Ctx:           ctx,
			Client:        client,
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
