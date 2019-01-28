package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvInt(t *testing.T) {
	const defaultValue = 200
	os.Setenv("TEST_ENV_STRING", "blah")
	os.Setenv("TEST_ENV_INT", "232")

	assert.Equal(t, 232, GetEnvInt("TEST_ENV_INT", defaultValue))
	assert.Equal(t, defaultValue, GetEnvInt("TEST_ENV_STRING", defaultValue))
	assert.Equal(t, defaultValue, GetEnvInt("FAKE_ENV", defaultValue))
}
