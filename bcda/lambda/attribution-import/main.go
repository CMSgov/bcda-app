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

var (
	pool      *pgxpool.Pool
	s3Client  *s3.Client
	ssmClient *ssm.Client
	dbURL     string
	logger    *logrus.Entry
)

func init() {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")

	ctx := context.Background()

	logger := configureLogger(env, appName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Fatalf("failed to load default config: %v", err)
	}

	ssmClient = ssm.NewFromConfig(cfg)
	s3Client = s3.NewFromConfig(cfg)

	dbURL, err = bcdaaws.GetParameter(ctx, ssmClient, fmt.Sprintf("/bcda/%s/sensitive/api/DATABASE_URL", env))
	if err != nil {
		logger.Fatalf("failed to load DB URL: %v", err)
	}

	if err = os.Setenv("DATABASE_URL", dbURL); err != nil {
		logger.Fatalf("failed to set DATABASE_URL: %v", err)
	}

	pool = database.ConnectPool()
}

func main() {
	lambda.Start(attributionImportHandler)
}

func attributionImportHandler(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
	// Reuse package-level logger with per-invocation fields if needed

	s3Event, err := bcdaaws.ParseSQSEvent(sqsEvent)
	if err != nil {
		logger.Errorf("failed to parse S3 event: %v", err)
		return "", err
	} else if s3Event == nil {
		logger.Info("No S3 event found, skipping safely.")
		return "", nil
	}

	var results []string
	var errs []error

	for _, e := range s3Event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			filepath := fmt.Sprintf("%s/%s", e.S3.Bucket.Name, e.S3.Object.Key)
			logger.Infof("Reading %s event for file %s", e.EventName, filepath)

			var result string
			if cclf.CheckIfAttributionCSVFile(e.S3.Object.Key) {
				result, err = handleCSVImport(ctx, pool, s3Client, filepath)
			} else {
				result, err = handleCclfImport(ctx, pool, s3Client, filepath)
			}
		}
	}

	logger.Info("No S3 ObjectCreated events found, skipping safely.")
	return "", nil
}

func handleCSVImport(ctx context.Context, pool *pgxpool.Pool, s3Client bcdaaws.CustomS3Client, s3ImportPath string) (string, error) {
	logger = logger.WithFields(logrus.Fields{"import_filename": s3ImportPath})

	err := loadBCDAParams()
	if err != nil {
		return "", err
	}

	importer := cclf.CSVImporter{
		Logger:  logger,
		PgxPool: pool,
		FileProcessor: &cclf.S3FileProcessor{
			Handler: optout.S3FileHandler{
				Client: s3Client,
				Logger: logger,
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

func handleCclfImport(ctx context.Context, pool *pgxpool.Pool, s3Client bcdaaws.CustomS3Client, s3ImportPath string) (string, error) {
	logger = logger.WithFields(logrus.Fields{"import_filename": s3ImportPath})

	err := loadBCDAParams()
	if err != nil {
		return "", err
	}

	fileProcessor := cclf.S3FileProcessor{
		Handler: optout.S3FileHandler{
			Client: s3Client,
			Logger: logger,
		},
	}

	importer := cclf.NewCclfImporter(logger, &fileProcessor, pool)
	success, failure, skipped, err := importer.ImportCCLFDirectory(s3ImportPath)
	if err != nil {
		logger.Error("error returned from ImportCCLFDirectory: ", err)
		return "", err
	}

	if failure > 0 || skipped > 0 {
		result := fmt.Sprintf("Successfully imported Attribution %v files.  Failed to import Attribution %v files.  Skipped %v Attribution files.  See logs for more details.", success, failure, skipped)
		logger.Error(result)

		err = errors.New("files skipped or failed import. See logs for more details")
		return result, err
	}

	result := fmt.Sprintf("Completed Attribution import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
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
