package main

import (
	"fmt"
	"os"
	"time"

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
		res, err := optOutImportHandler()
		if err != nil {
			fmt.Errorf("Failed to run opt out import: %s\n", err.Error())
		} else {
			fmt.Println(res)
		}
	} else {
		lambda.Start(optOutImportHandler)
	}
}

func optOutImportHandler() (string, error) {
	s3AssumeRoleArn, s3ImportPath, err := loadBfdS3Params()
	if err != nil {
		return "", err
	}

	return handleOptOutImport(s3AssumeRoleArn, s3ImportPath)
}

func loadBfdS3Params() (string, string, error) {
	env := conf.GetEnv("ENV")

	bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return "", "", err
	}

	s3AssumeRoleArnKey := fmt.Sprintf("/opt-out-import/bcda/%s/bfd-bucket-role-arn", env)
	s3ImportPathKey := fmt.Sprintf("/opt-out-import/bcda/%s/bfd-s3-import-path", env) // TODO

	params, err := bcdaaws.GetParameters(bcdaSession, []*string{&s3AssumeRoleArnKey, &s3ImportPathKey})
	if err != nil {
		return "", "", err
	}

	return params[s3AssumeRoleArnKey], params[s3ImportPathKey], nil
}

func handleOptOutImport(s3AssumeRoleArn, s3ImportPath string) (string, error) {
	env := conf.GetEnv("ENV")
	version := conf.GetEnv("APP_NAME")
	logger := configureLogger(env, version)
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
