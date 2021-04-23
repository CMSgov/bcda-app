package log

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestLoggers verifies that all of our loggers are set up
// with the expected parameters.
func TestLoggers(t *testing.T) {
	env := uuid.New()
	conf.SetEnv(t, "ENVIRONMENT", env)

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
			logFile, err := ioutil.TempFile("", "*")
			old := conf.GetEnv(tt.logEnv)
			t.Cleanup(func() {
				assert.NoError(t, os.Remove(logFile.Name()))
				assert.NoError(t, conf.SetEnv(t, tt.logEnv, old))
			})

			assert.NoError(t, err)
			conf.SetEnv(t, tt.logEnv, logFile.Name())

			// Refresh the logger to reference the new configs
			setup()

			msg := uuid.New()
			tt.logSupplier().Info(msg)
			tt.verify(t, env, msg, logFile)
		})
	}
}

func verifyAPILogs(t *testing.T, env, msg string, logFile *os.File) {
	data, err := ioutil.ReadAll(logFile)
	assert.NoError(t, err)

	res := strings.Split(string(data), "\n")
	// msg + new line
	assert.Len(t, res, 2)
	var fields logrus.Fields
	assert.NoError(t, json.Unmarshal([]byte(res[0]), &fields))
	assert.Equal(t, fields["msg"], msg)
	assert.Equal(t, fields["environment"], env)
	assert.Equal(t, fields["application"], "api")
	assert.Equal(t, fields["version"], constants.Version)
}

func verifyWorkerLogs(t *testing.T, env, msg string, logFile *os.File) {
	data, err := ioutil.ReadAll(logFile)
	assert.NoError(t, err)

	res := strings.Split(string(data), "\n")
	// msg + new line
	assert.Len(t, res, 2)
	var fields logrus.Fields
	assert.NoError(t, json.Unmarshal([]byte(res[0]), &fields))
	assert.Equal(t, fields["msg"], msg)
	assert.Equal(t, fields["environment"], env)
	assert.Equal(t, fields["application"], "worker")
	assert.Equal(t, fields["version"], constants.Version)
}
