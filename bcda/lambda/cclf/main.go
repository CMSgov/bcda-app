package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func main() {
	lambda.Start(attributionImportHandler)
}

func attributionImportHandler(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
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
	s3Client := s3.NewFromConfig(cfg)

	// Create pgx pool for bulk operations
	pool := database.ConnectPool()
	defer pool.Close()

	for _, e := range s3Event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			// Send the entire filepath into the CCLF Importer so we are only
			// importing the one file that was sent in the trigger.
			filepath := fmt.Sprintf("%s/%s", e.S3.Bucket.Name, e.S3.Object.Key)
			logger.Infof("Reading %s event for file %s", e.EventName, filepath)
			if cclf.CheckIfAttributionCSVFile(e.S3.Object.Key) {
				return handleCSVImport(ctx, pool, s3Client, ssmClient, filepath)
			} else {
				return handleCclfImport(ctx, pool, s3Client, ssmClient, filepath)
			}
		}
	}

	logger.Info("No ObjectCreated events found, skipping safely.")
	return "", nil
}

func handleCSVImport(ctx context.Context, pool *pgxpool.Pool, s3Client *s3.Client, ssmClient *ssm.Client, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	logger = logger.WithFields(logrus.Fields{"import_filename": s3ImportPath})

	err := loadBCDAParams()
	if err != nil {
		return "", err
	}

	s3AssumeRoleArn, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/cclf-import/bcda/%s/bfd-bucket-role-arn", env))
	if err != nil {
		logger.Errorf("error getting param: %+v", err)
		return "", err
	}

	importer := cclf.CSVImporter{
		Logger:  logger,
		PgxPool: pool,
		FileProcessor: &cclf.S3FileProcessor{
			Handler: optout.S3FileHandler{
				Client:        s3Client,
				Logger:        logger,
				Endpoint:      os.Getenv("LOCAL_STACK_ENDPOINT"),
				AssumeRoleArn: s3AssumeRoleArn,
			},
		},
	}

	err = importer.ImportCSV(ctx, s3ImportPath)
	if err != nil {
		logger.Error("error returned from ImportCSV: ", err)
		return "", err
	}

	result := fmt.Sprintf("Completed CSV import.  Successfully imported %v.   See logs for more details.", s3ImportPath)
	logger.Info(result)

	return result, nil
}

func handleCclfImport(ctx context.Context, pool *pgxpool.Pool, s3Client *s3.Client, ssmClient *ssm.Client, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)
	logger = logger.WithFields(logrus.Fields{"import_filename": s3ImportPath})

	err := loadBCDAParams()
	if err != nil {
		return "", err
	}

	s3AssumeRoleArn, err := bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/cclf-import/bcda/%s/bfd-bucket-role-arn", env))
	if err != nil {
		logger.Errorf("error getting param: %+v", err)
		return "", err
	}

	fileProcessor := cclf.S3FileProcessor{
		Handler: optout.S3FileHandler{
			Client:        s3Client,
			Logger:        logger,
			Endpoint:      os.Getenv("LOCAL_STACK_ENDPOINT"),
			AssumeRoleArn: s3AssumeRoleArn,
		},
	}

	importer := cclf.NewCclfImporter(logger, &fileProcessor, pool)
	success, failure, skipped, err := importer.ImportCCLFDirectory(s3ImportPath)
	if err != nil {
		logger.Error("error returned from ImportCCLFDirectory: ", err)
		return "", err
	}

	if failure > 0 || skipped > 0 {
		result := fmt.Sprintf("Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
		logger.Error(result)

		err = errors.New("files skipped or failed import. See logs for more details")
		return result, err
	}

	result := fmt.Sprintf("Completed CCLF import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
	logger.Info(result)

	return result, nil
}

func loadBCDAParams() error {
	env := conf.GetEnv("ENV")
	conf.LoadLambdaEnvVars(env)
	return nil
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
