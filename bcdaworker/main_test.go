package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/stretchr/testify/assert"
)

func TestClearTempDirectory(t *testing.T) {
	tempDirPrefix := conf.GetEnv("FHIR_TEMP_DIR")
	tempDir, err := os.MkdirTemp(tempDirPrefix, "bananas")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err)

	nestedDir := filepath.Join(tempDir, "nested")
	err = os.Mkdir(nestedDir, 0755)
	assert.DirExists(t, nestedDir)
	assert.NoError(t, err)

	tempFile := filepath.Join(tempDir, "tmp.txt")
	err = os.WriteFile(tempFile, []byte("test data"), 0644)
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
