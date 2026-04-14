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
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
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

			if err != nil {
				errs = append(errs, err)
			} else {
				results = append(results, result)
			}
		}
	}

	if len(errs) > 0 {
		return strings.Join(results, "; "), errors.Join(errs...)
	}

	if len(results) == 0 {
		logger.Info("No S3 ObjectCreated events found, skipping safely.")
	}

	return strings.Join(results, "; "), nil
}
