package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestLoggers_ToSTDOut(t *testing.T) {
	env := uuid.New()
	conf.SetEnv(t, "DEPLOYMENT_TARGET", env)
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
