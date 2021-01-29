package auth

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

type event struct {
	clientID   string
	elapsed    time.Duration
	help       string
	op         string
	tokenID    string
	trackingID string
}

// maybe we should just use plain standard logging and pass in json. We don't use the level
// designation to tune logging in non-local envs, so is logrus complexity worth it?
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
	mergeNonEmpty(data).WithField("event", "OperationStarted").Print(data.help)
}

func operationSucceeded(data event) {
	mergeNonEmpty(data).WithField("event", "OperationSucceeded").Print(data.help)
}

func operationFailed(data event) {
	mergeNonEmpty(data).WithField("event", "OperationFailed").Print(data.help)
}

func accessTokenIssued(data event) {
	mergeNonEmpty(data).WithField("event", "AccessTokenIssued").Print(data.help)
}

func secureHashTime(data event) {
	mergeNonEmpty(data).WithField("event", "SecureHashTime").Print(data.help)
}

func secretCreated(data event) {
	mergeNonEmpty(data).WithField("event", "SecretCreated").Print(data.help)
}

func serviceHalted(data event) {
	mergeNonEmpty(data).WithField("event", "ServiceHalted").Print(data.help)
}

func serviceStarted(data event) {
	mergeNonEmpty(data).WithField("event", "ServiceStarted").Print(data.help)
}
