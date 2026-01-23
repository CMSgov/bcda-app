package log

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
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

func TestDefaultLogger(t *testing.T) {
	API := defaultFieldLogger("test-log-type")
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

func TestSetLoggerFields(t *testing.T) {
	apiLogger := defaultFieldLogger("test-log-type")
	testLogger := test.NewLocal(testUtils.GetLogger(apiLogger))
	newLogEntry := &StructuredLoggerEntry{Logger: apiLogger}
	ctx := context.WithValue(context.Background(), CtxLoggerKey, newLogEntry)
	_, logger := SetLoggerFields(ctx, logrus.Fields{"request_id": "123456", "cms_id": "A0000"})

	logger.WithField("test", "entry").Error("test-msg")
	entry := testLogger.LastEntry()

	assert.Equal(t, "test-msg", entry.Message)
	assert.Equal(t, "123456", entry.Data["request_id"])
	assert.Equal(t, "A0000", entry.Data["cms_id"])
	assert.Equal(t, "entry", entry.Data["test"])
}

func TestWriteErrorWithFields(t *testing.T) {
	apiLogger := defaultFieldLogger("test-log-type")
	testLogger := test.NewLocal(testUtils.GetLogger(apiLogger))
	newLogEntry := &StructuredLoggerEntry{Logger: apiLogger}
	ctx := context.WithValue(context.Background(), CtxLoggerKey, newLogEntry)

	resultCtx, resultLogger := WriteErrorWithFields(ctx, "test-msg", logrus.Fields{"key1": "val1", "key2": "val2"})
	entry := testLogger.LastEntry()

	assert.Equal(t, "test-msg", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])
	assert.Equal(t, "val2", entry.Data["key2"])
	assert.Equal(t, logrus.ErrorLevel, entry.Level)

	// verify logger retains fields
	resultLogger.Error("new-test")
	entry = testLogger.LastEntry()

	assert.Equal(t, "new-test", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])

	// verify logger set in ctx retains fields
	verifyLogger := GetCtxLogger(resultCtx)
	verifyLogger.Error("newest-test")
	entry = testLogger.LastEntry()

	assert.Equal(t, "newest-test", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])
}

func TestWriteWarnWithFields(t *testing.T) {
	apiLogger := defaultFieldLogger("test-log-type")
	testLogger := test.NewLocal(testUtils.GetLogger(apiLogger))
	newLogEntry := &StructuredLoggerEntry{Logger: apiLogger}
	ctx := context.WithValue(context.Background(), CtxLoggerKey, newLogEntry)

	resultCtx, resultLogger := WriteWarnWithFields(ctx, "test-msg", logrus.Fields{"key1": "val1", "key2": "val2"})
	entry := testLogger.LastEntry()

	assert.Equal(t, "test-msg", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])
	assert.Equal(t, "val2", entry.Data["key2"])
	assert.Equal(t, logrus.WarnLevel, entry.Level)

	// verify logger retains fields
	resultLogger.Error("new-test")
	entry = testLogger.LastEntry()

	assert.Equal(t, "new-test", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])

	// verify logger set in ctx retains fields
	verifyLogger := GetCtxLogger(resultCtx)
	verifyLogger.Error("newest-test")
	entry = testLogger.LastEntry()

	assert.Equal(t, "newest-test", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])
}

func TestWriteInfoWithFields(t *testing.T) {
	apiLogger := defaultFieldLogger("test-log-type")
	testLogger := test.NewLocal(testUtils.GetLogger(apiLogger))
	newLogEntry := &StructuredLoggerEntry{Logger: apiLogger}
	ctx := context.WithValue(context.Background(), CtxLoggerKey, newLogEntry)

	resultCtx, resultLogger := WriteInfoWithFields(ctx, "test-msg", logrus.Fields{"key1": "val1", "key2": "val2"})
	entry := testLogger.LastEntry()

	assert.Equal(t, "test-msg", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])
	assert.Equal(t, "val2", entry.Data["key2"])
	assert.Equal(t, logrus.InfoLevel, entry.Level)

	// verify logger retains fields
	resultLogger.Error("new-test")
	entry = testLogger.LastEntry()

	assert.Equal(t, "new-test", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])

	// verify logger set in ctx retains fields
	verifyLogger := GetCtxLogger(resultCtx)
	verifyLogger.Error("newest-test")
	entry = testLogger.LastEntry()

	assert.Equal(t, "newest-test", entry.Message)
	assert.Equal(t, "val1", entry.Data["key1"])
}

func TestSlogLogger(t *testing.T) {
	oldEnvironment := conf.GetEnv("DEPLOYMENT_TARGET")
	environment := uuid.New()
	conf.SetEnv(t, "DEPLOYMENT_TARGET", environment)
	t.Cleanup(func() { conf.SetEnv(t, "DEPLOYMENT_TARGET", oldEnvironment) })

	application := "test_app"

	var output bytes.Buffer
	logger := slogLoggerFromHandler(slog.NewJSONHandler(&output, nil), application)
	logger.Info("test message")
	var logJson map[string]string
	err := json.Unmarshal(output.Bytes(), &logJson)

	assert.Nil(t, err)
	assert.Equal(t, "test message", logJson["msg"])
	assert.Equal(t, application, logJson["application"])
	assert.Equal(t, environment, logJson["environment"])
	assert.Equal(t, "bcda", logJson["source_app"])
	assert.Equal(t, constants.Version, logJson["version"])
}
