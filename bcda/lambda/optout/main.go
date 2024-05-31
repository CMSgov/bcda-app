package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/jackc/pgx/stdlib"
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
		optOutImportHandler()
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
	// fmt.Fprintf(app.Writer, "Completed 1-800-MEDICARE suppression data import.\nFiles imported: %v\nFiles failed: %v\nFiles skipped: %v\n", s, f, sk)
	logger.Infof("Completed 1-800-MEDICARE suppression data import.\nFiles imported: %v\nFiles failed: %v\nFiles skipped: %v\n", s, f, sk)
	return "success", err // TODO
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

func createDB(databaseUrl string) (*sql.DB, error) {
	dc := stdlib.DriverConfig{
		ConnConfig: pgx.ConnConfig{
			Logger:   logrusadapter.NewLogger(logrus.StandardLogger()),
			LogLevel: pgx.LogLevelError,
		},
		AfterConnect: func(c *pgx.Conn) error {
			// Can be used to ensure temp tables, indexes, etc. exist
			return nil
		},
	}

	stdlib.RegisterDriverConfig(&dc)

	db, err := sql.Open("nrpgx", dc.ConnectionString(databaseUrl))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
