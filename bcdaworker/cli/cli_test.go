package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/stretchr/testify/assert"
)

func TestClearTempDirectory(t *testing.T) {
	createWorkerDirs()
	tempDirPrefix := conf.GetEnv("FHIR_TEMP_DIR")
	tempDir, err := os.MkdirTemp(tempDirPrefix, "bananas")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err)

	nestedDir := filepath.Join(tempDir, "nested")
	err = os.Mkdir(nestedDir, 0755)
	assert.DirExists(t, nestedDir)
	assert.NoError(t, err)

	tempFile := filepath.Join(tempDir, "tmp.txt")
	err = os.WriteFile(tempFile, []byte("test data"), 0600)
	assert.NoError(t, err)

	dir, err := os.ReadDir(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(dir)) //We've created a file/directory, so we expect something.

	err = clearTempDirectory(tempDir)
	assert.NoError(t, err)

	dir, err = os.ReadDir(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(dir)) //Expect that there will be no entries after cleaning.

	err = clearTempDirectory("/fakedir") //Expect an error when there's an incorrect directory passed
	assert.Error(t, err)
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name            string
		dbOk            bool
		bbOk            bool
		expectedHealthy bool
	}{
		{"Database and BlueButton healthy", true, true, true},
		{"Database unhealthy", false, true, false},
		{"BlueButton unhealthy", true, false, false},
		{"Database and BlueButton unhealthy", false, false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockHealthChecker := &health.MockHealthChecker{}
			mockHealthChecker.On("IsWorkerDatabaseOK").Return("", test.dbOk)
			mockHealthChecker.On("IsBlueButtonOK").Return(test.bbOk)
			actualHealthy := checkHealth(mockHealthChecker)
			assert.Equal(t, test.expectedHealthy, actualHealthy)
		})
	}
}
