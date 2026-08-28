package filehandler

import (
	"github.com/sirupsen/logrus"
)

// LocalFileHandler manages generic file operations from local directories.
type LocalFileHandler struct {
	Logger                 logrus.FieldLogger
	PendingDeletionDir     string
	FileArchiveThresholdHr uint
}

func (handler *LocalFileHandler) Infof(format string, rest ...interface{}) {
	if handler.Logger != nil {
		handler.Logger.Infof(format, rest...)
	}
}

func (handler *LocalFileHandler) Warningf(format string, rest ...interface{}) {
	if handler.Logger != nil {
		handler.Logger.Warningf(format, rest...)
	}
}

func (handler *LocalFileHandler) Errorf(format string, rest ...interface{}) {
	if handler.Logger != nil {
		handler.Logger.Errorf(format, rest...)
	}
}
