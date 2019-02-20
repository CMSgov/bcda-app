package client

import (
	"os"

	"github.com/pborman/uuid"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	filePath, success := os.LookupEnv("BCDA_OKTA_LOG")
	if success {
		/* #nosec -- 0640 permissions required for Splunk ingestion */
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.Info("Failed to open Okta log file; using default stderr")
		}
	} else {
		logger.Info("No Okta log location provided; using default stderr")
	}

	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")
	if oktaToken == "" {
		logger.Fatal("No Okta token found; please set OKTA_CLIENT_TOKEN")
	}

	oktaURL := os.Getenv("OKTA_CLIENT_ORGURL")
	if oktaURL == "" {
		logger.Fatal("No Okta URL found; please set OKTA_CLIENT_ORGURL")
	}
}

func logRequest(requestId uuid.UUID) *logrus.Entry {
	return logger.WithField("request_id", requestId)
}

func logResponse(httpStatus int, requestId uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"http_status": httpStatus, "request_id": requestId})
}

func logError(err error, requestId uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"error": err, "request_id": requestId})
}
