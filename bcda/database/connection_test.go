package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestConnections(t *testing.T) {
	// Verify that we can initialize the package as expected
	assert.NotNil(t, Connection)
	assert.NotNil(t, QueueConnection)

	assert.NoError(t, Connection.Ping())
	c, err := QueueConnection.Acquire()
	assert.NoError(t, err)
	assert.NoError(t, c.Ping(context.Background()))
	QueueConnection.Release(c)
}

// TestHealthCheck verifies that we are able to start the health check
// and the checks do not cause a panic by waiting some amount of time
// to ensure that health checks are being executed.
func TestHealthCheck(t *testing.T) {
	level, reporter := logrus.GetLevel(), logrus.StandardLogger().ReportCaller
	t.Cleanup(func() {
		logrus.SetLevel(level)
		logrus.SetReportCaller(reporter)
	})

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	hook := test.NewGlobal()

	ctx, cancel := context.WithCancel(context.Background())
	startHealthCheck(ctx, Connection, QueueConnection, 100*time.Microsecond)
	// Let some time elapse to ensure we've successfully ran health checks
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	var hasPing, hasClose bool
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Caller.File, "database/connection.go") {
			if strings.Contains(entry.Message, "Sending ping") {
				hasPing = true
			} else if strings.Contains(entry.Message, "Stopping health checker") {
				hasClose = true
			}
		}
	}

	assert.True(t, hasPing, "Should've received a ping message in the logs.")
	assert.True(t, hasClose, "Should've received a close message in the logs.")
}
