package log

import (
	"os"
	"path/filepath"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/sirupsen/logrus"
)

var (
	API     logrus.FieldLogger
	Auth    logrus.FieldLogger
	BBAPI   logrus.FieldLogger
	Request logrus.FieldLogger
	SSAS    logrus.FieldLogger

	Worker   logrus.FieldLogger
	BBWorker logrus.FieldLogger
)

func init() {
	API = Logger(logrus.New(), conf.GetEnv("BCDA_ERROR_LOG"),
		"api", conf.GetEnv("ENVIRONMENT"))
	Auth = Logger(logrus.New(), conf.GetEnv("AUTH_LOG"),
		"api", conf.GetEnv("ENVIRONMENT"))
	BBAPI = Logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"),
		"api", conf.GetEnv("ENVIRONMENT"))
	Request = Logger(logrus.New(), conf.GetEnv("BCDA_REQUEST_LOG"),
		"api", conf.GetEnv("ENVIRONMENT"))
	SSAS = Logger(logrus.New(), conf.GetEnv("BCDA_SSAS_LOG"),
		"api", conf.GetEnv("ENVIRONMENT"))

	Worker = Logger(logrus.New(), conf.GetEnv("BCDA_WORKER_ERROR_LOG"),
		"worker", conf.GetEnv("ENVIRONMENT"))
	BBWorker = Logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"),
		"worker", conf.GetEnv("ENVIRONMENT"))
}

func Logger(logger *logrus.Logger, outputFile string,
	application, environment string) logrus.FieldLogger {

	if outputFile != "" {
		if file, err := os.OpenFile(filepath.Clean(outputFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
			logger.SetOutput(file)
		} else {
			logger.Infof("Failed to open output file %s. Will use stderr. %s",
				outputFile, err.Error())
		}
	}

	return logger.WithFields(logrus.Fields{
		"application": application,
		"environment": environment})
}
