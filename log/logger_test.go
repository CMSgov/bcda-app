package log

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
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
		{"BCDA_BB_LOG", func() logrus.FieldLogger { return BBAPI }, verifyAPILogs},
		{"BCDA_REQUEST_LOG", func() logrus.FieldLogger { return Request }, verifyAPILogs},
		{"BCDA_SSAS_LOG", func() logrus.FieldLogger { return SSAS }, verifyAPILogs},

		{"BCDA_WORKER_ERROR_LOG", func() logrus.FieldLogger { return Worker }, verifyWorkerLogs},
		{"BCDA_BB_LOG", func() logrus.FieldLogger { return BBWorker }, verifyWorkerLogs},
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
	assert.Equal(t, fields["environment"], env)
	assert.Equal(t, fields["msg"], msg)
	assert.Equal(t, fields["version"], constants.Version)
	_, err := time.Parse(time.RFC3339Nano, fields["time"].(string))
	assert.NoError(t, err)
}
