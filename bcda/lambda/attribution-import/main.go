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

	ai "github.com/CMSgov/bcda-app/bcda/attribution-import"
	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type AttributionImportHandler struct {
	logger       *logrus.Entry
	pool         *pgxpool.Pool
	fileClient   bcdaaws.CustomS3Client
	csvImporter  ai.CSVImporterInterface
	cclfImporter ai.CCLFImporterInterface
	checkIfCSV   func(filePath string) (bool, error)
}

func main() {
	ctx := context.Background()
	handler, err := initHandler(ctx)
	if err != nil {
		logrus.Fatalf("failed to initialize handler: %v", err)
	}
	if handler.pool != nil {
		defer handler.pool.Close()
	}

	lambda.Start(handler.handleImport)
}

func initHandler(ctx context.Context) (*AttributionImportHandler, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("failed to load Default Config")
		return nil, err
	}

	ssmClient := ssm.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

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

	pool := database.ConnectPool()

	err = loadBCDAParams()
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, err
	}

	handler := &AttributionImportHandler{
		logger:     logger,
		pool:       pool,
		fileClient: s3Client,
	}

	return handler, nil
}

func (h *AttributionImportHandler) handleImport(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
	if len(sqsEvent.Records) == 0 {
		h.logger.Info("No SQS records found, skipping safely.")
		return "", nil
	}

	s3Event, err := bcdaaws.ParseSQSEvent(sqsEvent)
	if err != nil {
		h.logger.Errorf("failed to parse S3 event: %v", err)
		return "", err
	} else if s3Event == nil {
		h.logger.Info("No S3 event found, skipping safely.")
		return "", nil
	}

	checkCSV := h.checkIfCSV
	if checkCSV == nil {
		checkCSV = ai.CheckIfAttributionCSVFile
	}

	// SQS event messages from S3 notifications are dispatched per object creation.
	// Process the first ObjectCreated record found in the SQS event batch.
	for _, e := range s3Event.Records {
		if strings.Contains(e.EventName, "ObjectCreated") {
			filepath := fmt.Sprintf("%s/%s", e.S3.Bucket.Name, e.S3.Object.Key)
			h.logger.Infof("Reading %s event for file %s", e.EventName, filepath)
			isCSV, err := checkCSV(e.S3.Object.Key)
			if err != nil {
				h.logger.Errorf("error checking if file is CSV: %v", err)
				return "", err
			} else if isCSV {
				return h.handleCSVImport(ctx, filepath)
			} else {
				return h.handleCclfImport(ctx, filepath)
			}
		}
	}

	h.logger.Info("No S3 ObjectCreated events found, skipping safely.")
	return "", nil
}

func (h *AttributionImportHandler) handleCSVImport(ctx context.Context, s3ImportPath string) (string, error) {
	logger := h.logger.WithFields(logrus.Fields{"import_filename": s3ImportPath})

	importer := h.csvImporter
	if importer == nil {
		importer = ai.CSVImporter{
			Logger:     logger,
			PgxPool:    h.pool,
			FileClient: h.fileClient,
		}
	}

	err := importer.ImportCSV(ctx, s3ImportPath)
	if err != nil {
		logger.WithError(err).Error("error returned from ImportCSV")
		return "", err
	}

	result := fmt.Sprintf("Completed CSV import.  Successfully imported %v.   See logs for more details.", s3ImportPath)
	logger.Info(result)

	return result, nil
}

func (h *AttributionImportHandler) handleCclfImport(ctx context.Context, s3ImportPath string) (string, error) {
	logger := h.logger.WithFields(logrus.Fields{"import_filename": s3ImportPath})

	importer := h.cclfImporter
	if importer == nil {
		importer = ai.NewCCLFImporter(logger, h.fileClient, h.pool)
	}

	success, failure, skipped, err := importer.ImportCCLFDirectory(ctx, s3ImportPath)
	if err != nil {
		logger.WithError(err).Error("error returned from ImportCCLFDirectory")
		return "", err
	}

	if failure > 0 || skipped > 0 {
		result := fmt.Sprintf("Successfully imported Attribution %v files.  Failed to import Attribution %v files.  Skipped %v Attribution files.  See logs for more details.", success, failure, skipped)
		logger.Error(result)

		return result, errors.New("files skipped or failed import. See logs for more details")
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
