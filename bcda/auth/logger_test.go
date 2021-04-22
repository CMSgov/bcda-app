package auth

import (
	"testing"

	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestOperationLogging(t *testing.T) {
	testLogger := test.NewLocal(log.Auth)
	operationStarted(event{op: "TestOperation", help: "A little more to the right"})

	assert.Equal(t, 1, len(testLogger.Entries))
	assert.Equal(t, logrus.InfoLevel, testLogger.LastEntry().Level)
	assert.Equal(t, "A little more to the right", testLogger.LastEntry().Message)

	testLogger.Reset()
	assert.Nil(t, testLogger.LastEntry())
}
