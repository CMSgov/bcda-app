package main

import (
	"database/sql"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

func NewHealthLogger() *HealthLogger {
	l := HealthLogger{logrus.New()}
	l.Logger.Formatter = &logrus.JSONFormatter{}
	filePath := os.Getenv("WORKER_HEALTH_LOG")

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
	logFields["db"] = isDatabaseOK()
	logFields["bb"] = isBlueButtonOK()
	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Info()
}

type HealthLogEntry struct {
	Logger logrus.FieldLogger
}

// TODO: Share with API instead of duplicating database check logic
func isDatabaseOK() bool {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return false
	}
	defer db.Close()

	return db.Ping() == nil
}

func isBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient()
	if err != nil {
		return false
	}

	_, err = bbc.GetMetadata()
	return err == nil
}
