package main

import (
	"os"

	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

func NewHealthLogger() *HealthLogger {
	l := HealthLogger{logrus.New()}
	l.Logger.Formatter = &logrus.JSONFormatter{}
	l.Logger.SetReportCaller(true)
	filePath := conf.GetEnv("WORKER_HEALTH_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		l.Logger.SetOutput(file)
	} else {
		l.Logger.Info("Failed to open worker health log file; using default stderr")
	}

	return &l
}

type HealthLogger struct {
	Logger *logrus.Logger
}

func (l *HealthLogger) Log() {
	entry := &HealthLogEntry{Logger: logrus.NewEntry(l.Logger)}

	logFields := logrus.Fields{}
	logFields["type"] = "health"
	logFields["id"] = uuid.NewRandom()

	if health.IsDatabaseOK() {
		logFields["db"] = "ok"
	} else {
		logFields["db"] = "error"
	}

	if health.IsBlueButtonOK() {
		logFields["bb"] = "ok"
	} else {
		logFields["bb"] = "error"
	}

	entry.Logger.WithFields(logFields).Info()
}

type HealthLogEntry struct {
	Logger logrus.FieldLogger
}
