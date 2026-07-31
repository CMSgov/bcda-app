package worker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDir(t *testing.T) {
	err := CreateDir("testdir")
	assert.NoError(t, err)
	assert.DirExists(t, "testdir")

	// run it again for idempotency
	err = CreateDir("testdir")
	assert.NoError(t, err)
	assert.DirExists(t, "testdir")
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
	assert.Contains(t, string(content), "Hello, World!\n")

	// repeat to append more data to end of file
	err = AppendToFile(filePath, []byte("Another entry!\n"))
	assert.NoError(t, err)

	content, err = os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Hello, World!\nAnother entry!\n")

	err = os.Remove(filePath)
	assert.NoError(t, err)
}
