package worker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDir(t *testing.T) {
	dir := t.TempDir()
	testDir := dir + "/testdir"

	err := CreateDir(testDir)
	assert.NoError(t, err)
	assert.DirExists(t, testDir)

	// run it again for idempotency
	err = CreateDir(testDir)
	assert.NoError(t, err)
	assert.DirExists(t, testDir)
}

func TestAppendToFile(t *testing.T) {
	filePath := os.TempDir() + "/testfile.txt"
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	assert.NoError(t, err)
	defer file.Close()

	err = AppendToFile(filePath, []byte("Hello, World!"))
	assert.NoError(t, err)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Hello, World!")

	// repeat to append more data to end of file
	err = AppendToFile(filePath, []byte("Another entry!"))
	assert.NoError(t, err)

	content, err = os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Hello, World!Another entry!")

	err = os.Remove(filePath)
	assert.NoError(t, err)
}
