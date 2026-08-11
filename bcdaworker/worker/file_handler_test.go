package worker

import (
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
