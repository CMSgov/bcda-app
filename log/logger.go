package log

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

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
