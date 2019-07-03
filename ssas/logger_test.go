package ssas

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestOperationLogging(t*testing.T){
	testLogger := test.NewLocal(logger)
	OperationStarted(Event{Op: "TestOperation", Help: "A little more to the right"})

	assert.Equal(t, 1, len(testLogger.Entries))
	assert.Equal(t, logrus.InfoLevel, testLogger.LastEntry().Level)
	assert.Equal(t, "A little more to the right", testLogger.LastEntry().Message)

	testLogger.Reset()
	assert.Nil(t, testLogger.LastEntry())
}