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

/*
	The following logging functions should be passed an Event{} with at least the Op and TrackingID set, and
	other general messages put in the Help field.  Successive logs for the same event should use the same
	randomly generated TrackingID.
 */

/*
	OperationStarted should be called at the beginning of all logged events
 */
func OperationStarted(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationStarted").Print(data.Help)
}

/*
	OperationSucceeded should be called after an event's success, and should always be preceded by
	a call to OperationStarted
 */
func OperationSucceeded(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationSucceeded").Print(data.Help)
}

/*
	OperationCalled will log the caller of an operation.  The caller should use the same
	randomly generated TrackingID as used in the operation for OperationStarted, OperationSucceeded, etc.
 */
func OperationCalled(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationCalled").Print(data.Help)
}

/*
	OperationFailed should be called after an event's failure, and should always be preceded by
	a call to OperationStarted
 */
func OperationFailed(data Event) {
	mergeNonEmpty(data).WithField("Event", "OperationFailed").Print(data.Help)
}

/*
	AccessTokenIssued should be called to log the successful creation of every access token
 */
func AccessTokenIssued(data Event) {
	mergeNonEmpty(data).WithField("Event", "AccessTokenIssued").Print(data.Help)
}

/*
	SecureHashTime should be called with the time taken to create a hash, logs of which can be used
	to approximate the security provided by the hash
 */
func SecureHashTime(data Event) {
	mergeNonEmpty(data).WithField("Event", "SecureHashTime").Print(data.Help)
}

/*
	SecretCreated should be called every time a system's secret is generated
 */
func SecretCreated(data Event) {
	mergeNonEmpty(data).WithField("Event", "SecretCreated").Print(data.Help)
}

/*
	ServiceHalted should be called to log an unexpected stop to the service
 */
func ServiceHalted(data Event) {
	mergeNonEmpty(data).WithField("Event", "ServiceHalted").Print(data.Help)
}

/*
	ServiceStarted should be called to log the starting of the service
 */
func ServiceStarted(data Event) {
	mergeNonEmpty(data).WithField("Event", "ServiceStarted").Print(data.Help)
}