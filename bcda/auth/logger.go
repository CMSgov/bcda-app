package auth

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

const (
	// tknExpired       = "AccessTokenExpired"
	// tknInvalid       = "AccessTokenInvalid"
	// tknIssued        = "AccessTokenIssued"
	// tknRevoked       = "AccessTokenRevoked"
	// credsIssued      = "ClientCredentialsIssued"
	// credsRevoked     = "ClientCredentialsRevoked"
	// reqAuthorized    = "RequestAuthorized"
	// reqNotAuthorized = "RequestNotAuthorized"
	// requesterUnknown = "RequesterUnknown"
	// secretCreated    = "SecretCreated"
	// secretFast       = "SecretTooFast"
	// secretSlow       = "SecretTooSlow"
	// a failure of something other than auth logic: duplicate db entries, network failures
	// sysFailure = "SystemFailure"
	// operation events
	opFailed    = "OperationFailed"
	opStarted   = "OperationStarted"
	opSucceeded = "OperationSucceeded"
	// service events
	// halted  = "ServiceHalted"
	// started = "ServiceStarted"
	// stopped = "ServiceStopped"
)

type event struct {
	clientID   string
	elapsed    int
	help       string
	op         string
	tokenID    string
	trackingID string
}

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Formatter.(*logrus.JSONFormatter).TimestampFormat = time.RFC3339Nano

	filePath, success := os.LookupEnv("AUTH_LOG")
	if success {
		/* #nosec -- 0640 permissions required for Splunk ingestion */
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.Info("Failed to open Auth log file; using default stderr")
		}
	} else {
		logger.Info("No Auth log location provided; using default stderr")
	}
}

func mergeNonEmpty(data event) *logrus.Entry {
	var entry = logrus.NewEntry(logger)

	if data.clientID != "" {
		entry = entry.WithField("clientID", data.clientID)
	}
	if data.trackingID != "" {
		entry = entry.WithField("trackingID", data.trackingID)
	}
	if data.elapsed != 0 {
		entry = entry.WithField("elapsed", data.elapsed)
	}
	if data.op != "" {
		entry = entry.WithField("op", data.op)
	}
	if data.tokenID != "" {
		entry = entry.WithField("tokenID", data.tokenID)
	}

	return entry
}

func operationStarted(data event) {
	mergeNonEmpty(data).WithField("event", opStarted).Print(data.help)
}

func operationSucceeded(data event) {
	mergeNonEmpty(data).WithField("event", opSucceeded).Print(data.help)
}

func operationFailed(data event) {
	mergeNonEmpty(data).WithField("event", opFailed).Print(data.help)
}