package log

import (
	"os"
	"path/filepath"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
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
	Health   logrus.FieldLogger
)

func init() {
	setup()
}

// setup allows us to invoke it automatically (via init()) and in tests
// In tests, we want to set up the files/environment in a consistent manner
func setup() {
	env := conf.GetEnv("DEPLOYMENT_TARGET")

	API = logger(logrus.New(), conf.GetEnv("BCDA_ERROR_LOG"),
		"api", env)
	Auth = logger(logrus.New(), conf.GetEnv("AUTH_LOG"),
		"api", env)
	BBAPI = logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"),
		"api", env)
	Request = logger(logrus.New(), conf.GetEnv("BCDA_REQUEST_LOG"),
		"api", env)
	SSAS = logger(logrus.New(), conf.GetEnv("BCDA_SSAS_LOG"),
		"api", env)

	Worker = logger(logrus.New(), conf.GetEnv("BCDA_WORKER_ERROR_LOG"),
		"worker", env)
	BBWorker = logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"),
		"worker", env)
	Health = logger(logrus.New(), conf.GetEnv("WORKER_HEALTH_LOG"),
		"worker", env)
}

func logger(logger *logrus.Logger, outputFile string,
	application, environment string) logrus.FieldLogger {

	if outputFile != "" {
		// #nosec G302 -- 0640 permissions required for Splunk ingestion
		if file, err := os.OpenFile(filepath.Clean(outputFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
			logger.SetOutput(file)
		} else {
			logger.Infof("Failed to open output file %s. Will use stderr. %s",
				outputFile, err.Error())
		}
	}
	// Disable the HTML escape so we get the raw URLs
	logger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	logger.SetReportCaller(true)

	return logger.WithFields(logrus.Fields{
		"application": application,
		"environment": environment,
		"version":     constants.Version})
}
