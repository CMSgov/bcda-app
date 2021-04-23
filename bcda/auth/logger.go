package auth

import (
	"time"

	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
)

type event struct {
	clientID   string
	elapsed    time.Duration
	help       string
	op         string
	tokenID    string
	trackingID string
}

func mergeNonEmpty(data event) logrus.FieldLogger {
	var entry = log.Auth

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
