package log

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// TestLoggers verifies that all of our loggers are set up
// with the expected parameters and write to the expected files.
func TestLoggers(t *testing.T) {
	env := uuid.New()
	conf.SetEnv(t, "DEPLOYMENT_TARGET", env)

	tests := []struct {
		logEnv string
		// Use a supplier since the logger's reference will be updated everytime we call
		// setup func. This allows us to retrieve the refreshed logger
		logSupplier func() logrus.FieldLogger
		verify      func(*testing.T, string, string, *os.File)
	}{
		{"BCDA_ERROR_LOG", func() logrus.FieldLogger { return API }, verifyAPILogs},
		{"AUTH_LOG", func() logrus.FieldLogger { return Auth }, verifyAPILogs},
		{"BCDA_BB_LOG", func() logrus.FieldLogger { return BFDAPI }, verifyAPILogs},
		{"BCDA_REQUEST_LOG", func() logrus.FieldLogger { return Request }, verifyAPILogs},
		{"BCDA_SSAS_LOG", func() logrus.FieldLogger { return SSAS }, verifyAPILogs},

		{"BCDA_WORKER_ERROR_LOG", func() logrus.FieldLogger { return Worker }, verifyWorkerLogs},
		{"BCDA_BB_LOG", func() logrus.FieldLogger { return BFDWorker }, verifyWorkerLogs},
		{"WORKER_HEALTH_LOG", func() logrus.FieldLogger { return Health }, verifyWorkerLogs},
	}
	for _, tt := range tests {
		t.Run(tt.logEnv, func(t *testing.T) {
			logFile, err := os.CreateTemp("", "*")
			old := conf.GetEnv(tt.logEnv)
			t.Cleanup(func() {
				assert.NoError(t, os.Remove(logFile.Name()))
				assert.NoError(t, conf.SetEnv(t, tt.logEnv, old))
			})

			assert.NoError(t, err)
			conf.SetEnv(t, tt.logEnv, logFile.Name())

			// Refresh the logger to reference the new configs
			SetupLoggers()

			msg := uuid.New()
			tt.logSupplier().Info(msg)
			tt.verify(t, env, msg, logFile)
		})
	}
}

func verifyAPILogs(t *testing.T, env, msg string, logFile *os.File) {
	data, err := io.ReadAll(logFile)
	assert.NoError(t, err)

	res := strings.Split(string(data), "\n")
	// msg + new line
	assert.Len(t, res, 2)
	var fields logrus.Fields
	assert.NoError(t, json.Unmarshal([]byte(res[0]), &fields))
	assert.Equal(t, fields["application"], "api")
	verifyCommonFields(t, fields, env, msg)
}

func verifyWorkerLogs(t *testing.T, env, msg string, logFile *os.File) {
	data, err := io.ReadAll(logFile)
	assert.NoError(t, err)

	res := strings.Split(string(data), "\n")
	// msg + new line
	assert.Len(t, res, 2)
	var fields logrus.Fields
	assert.NoError(t, json.Unmarshal([]byte(res[0]), &fields))
	assert.Equal(t, fields["application"], "worker")
	verifyCommonFields(t, fields, env, msg)
}

func verifyCommonFields(t *testing.T, fields logrus.Fields, env, msg string) {
	assert.Equal(t, env, fields["environment"])
	assert.Equal(t, msg, fields["msg"])
	assert.Equal(t, "bcda", fields["source_app"])
	assert.Equal(t, constants.Version, fields["version"])
	_, err := time.Parse(time.RFC3339Nano, fields["time"].(string))
	assert.NoError(t, err)
}

func TestLoggers_ToSTDOut(t *testing.T) {
	env := uuid.New()
	conf.SetEnv(t, "DEPLOYMENT_TARGET", env)
	oldVal := conf.GetEnv("LOG_TO_STD_OUT")
	conf.SetEnv(t, "LOG_TO_STD_OUT", "true")
	t.Cleanup(func() { conf.SetEnv(t, "LOG_TO_STD_OUT", oldVal) })

	tests := []struct {
		logType string
		// Use a supplier since the logger's reference will be updated everytime we call
		// setup func. This allows us to retrieve the refreshed logger
		logSupplier func() logrus.FieldLogger
	}{
		{"api", func() logrus.FieldLogger { return API }},
		{"auth", func() logrus.FieldLogger { return Auth }},
		{"bfd", func() logrus.FieldLogger { return BFDAPI }},
		{"request", func() logrus.FieldLogger { return Request }},
		{"ssas", func() logrus.FieldLogger { return SSAS }},

		{"worker", func() logrus.FieldLogger { return Worker }},
		{"bfd", func() logrus.FieldLogger { return BFDWorker }},
		{"health", func() logrus.FieldLogger { return Health }},
	}
	for _, tt := range tests {
		t.Run(tt.logType, func(t *testing.T) {
			// Refresh the logger to reference the new configs
			SetupLoggers()

			testLogger := test.NewLocal(testUtils.GetLogger(tt.logSupplier()))

			msg := uuid.New()
			tt.logSupplier().Info(msg)

			assert.Equal(t, 1, len(testLogger.Entries))
			assert.Equal(t, msg, testLogger.LastEntry().Message)
			assert.Equal(t, tt.logType, testLogger.LastEntry().Data["log_type"])
			assert.Equal(t, "bcda", testLogger.LastEntry().Data["source_app"])
			testLogger.Reset()
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	API := defaultLogger("test-log-type")
	testLogger := test.NewLocal(testUtils.GetLogger(API))

	msg := uuid.New()
	API.Info(msg)

	assert.Equal(t, 1, len(testLogger.Entries))
	assert.Equal(t, msg, testLogger.LastEntry().Message)
	assert.Equal(t, "default", testLogger.LastEntry().Data["application"])
	assert.Equal(t, conf.GetEnv("DEPLOYMENT_TARGET"), testLogger.LastEntry().Data["environment"])
	assert.Equal(t, "test-log-type", testLogger.LastEntry().Data["log_type"])
	assert.Equal(t, constants.Version, testLogger.LastEntry().Data["version"])
}
