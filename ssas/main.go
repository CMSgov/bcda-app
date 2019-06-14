package main

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Formatter.(*logrus.JSONFormatter).TimestampFormat = time.RFC3339Nano
	filePath, success := os.LookupEnv("SSAS_LOG_FILE")
	if success {
		/* #nosec -- 0640 permissions required for Splunk ingestion */
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.Info("Failed to open SSAS log file; using default stderr")
		}
	} else {
		logger.Info("No SSAS log location provided; using default stderr")
	}
}

func main() {
	logger.Info("Future home of the System-to-System Authentication Service")
}

func hello() string {
	return "hello SSAS"
}

