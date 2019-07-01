package ssas

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// TODO: Remove auth.logger
var logger *logrus.Logger

type Event struct {
	ClientID   string
	Elapsed    time.Duration
	Help       string
	Op         string
	TokenID    string
	TrackingID string
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

func mergeNonEmpty(data Event) *logrus.Entry {
	var entry = logrus.NewEntry(logger)

	if data.ClientID != "" {
		entry = entry.WithField("clientID", data.ClientID)
	}
	if data.TrackingID != "" {
		entry = entry.WithField("trackingID", data.TrackingID)
	}
	if data.Elapsed != 0 {
		entry = entry.WithField("elapsed", data.Elapsed)
	}
	if data.Op != "" {
		entry = entry.WithField("op", data.Op)
	}
	if data.TokenID != "" {
		entry = entry.WithField("tokenID", data.TokenID)
	}

	return entry
}

func OperationStarted(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationStarted").Print(data.Help)
}

func OperationSucceeded(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationSucceeded").Print(data.Help)
}

func OperationCalled(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationCalled").Print(data.Help)
}

func OperationFailed(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationFailed").Print(data.Help)
}

func AccessTokenIssued(data Event) {
	mergeNonEmpty(data).WithField("Event", "AccessTokenIssued").Print(data.Help)
}

func SecureHashTime(data Event) {
	mergeNonEmpty(data).WithField("Event", "SecureHashTime").Print(data.Help)
}

func SecretCreated(data Event) {
	mergeNonEmpty(data).WithField("Event", "SecretCreated").Print(data.Help)
}

func ServiceHalted(data Event) {
	mergeNonEmpty(data).WithField("Event", "ServiceHalted").Print(data.Help)
}

func ServiceStarted(data Event) {
	mergeNonEmpty(data).WithField("Event", "ServiceStarted").Print(data.Help)
}