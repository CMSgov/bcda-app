package filehandler

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestLocalLoggerFunctions(t *testing.T) {
	logger, hook := test.NewNullLogger()
	handler := &LocalFileHandler{
		Logger: logger,
	}

	handler.Infof("test info message")
	assert.Equal(t, "test info message", hook.LastEntry().Message)

	handler.Warningf("test warning message")
	assert.Equal(t, "test warning message", hook.LastEntry().Message)

	handler.Errorf("test error message")
	assert.Equal(t, "test error message", hook.LastEntry().Message)
}

func TestLocalNilLoggerFunctions(t *testing.T) {
	handler := &LocalFileHandler{
		Logger: nil,
	}
	assert.NotPanics(t, func() {
		handler.Infof("info")
		handler.Warningf("warn")
		handler.Errorf("err")
	})
}
