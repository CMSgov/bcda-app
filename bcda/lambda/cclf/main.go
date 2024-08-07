package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/CMSgov/bcda-app/conf"
)

func main() {
	// Localstack is a local-development server that mimics AWS. The endpoint variable
	// should only be set in local development to avoid making external calls to a real AWS account.
	if os.Getenv("LOCAL_STACK_ENDPOINT") != "" {
		res, err := handleCclfImport(os.Getenv("BFD_BUCKET_ROLE_ARN"), os.Getenv("BFD_S3_IMPORT_PATH"))
		if err != nil {
			fmt.Printf("Failed to run opt out import: %s\n", err.Error())
		} else {
			fmt.Println(res)
		}
	} else {
		lambda.Start(cclfImportHandler)
	}
}

func cclfImportHandler(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
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
		logger.Info(e.EventName)
		if e.EventName == "ObjectCreated:Put" {
			s3AssumeRoleArn, err := loadBfdS3Params()
			if err != nil {
				return "", err
			}

			// Send the entire filepath into the CCLF Importer so we are only
			// importing the one file that was sent in the trigger.
			filepath := fmt.Sprintf("%s/%s", e.S3.Bucket.Name, e.S3.Object.Key)
			return handleCclfImport(s3AssumeRoleArn, filepath)
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

	param, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/cclf-import/bcda/%s/bfd-bucket-role-arn", env))
	if err != nil {
		return "", err
	}

	return param, nil
}

func handleCclfImport(s3AssumeRoleArn, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	appName := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, appName)

	importer := cclf.CclfImporter{
		FileProcessor: &cclf.S3FileProcessor{
			Handler: optout.S3FileHandler{
				Logger:        logger,
				Endpoint:      os.Getenv("LOCAL_STACK_ENDPOINT"),
				AssumeRoleArn: s3AssumeRoleArn,
			},
		},
	}

	success, failure, skipped, err := importer.ImportCCLFDirectory(s3ImportPath)

	if err != nil {
		logger.Error("error returned from ImportCCLFDirectory: ", err)
		return "", err
	}

	if failure > 0 || skipped > 0 {
		result := fmt.Sprintf("Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
		logger.Error(result)

		err = errors.New("Files skipped or failed import. See logs for more details.")
		return result, err

	}

	result := fmt.Sprintf("Completed CCLF import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
	logger.Info(result)
	return result, nil
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
