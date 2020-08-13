package metrics

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlogrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type MetricTestSuite struct {
	suite.Suite
	timer Timer
	hook  *test.Hook
}

func (s *MetricTestSuite) SetupTest() {
	s.hook = test.NewGlobal()

	logrus.SetLevel(logrus.DebugLevel)

	c := newrelic.Config{
		Enabled: false,
		Logger:  nrlogrus.StandardLogger(),
	}

	nr, err := newrelic.NewApplication(c)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), nr)

	// Reset the test hook so we ignore any logs associated
	// with the setup of NewRelic
	s.hook.Reset()
	s.timer = &timer{nr}
}

func TestMetricTestSuite(t *testing.T) {
	suite.Run(t, new(MetricTestSuite))
}

func (s *MetricTestSuite) TestContext() {
	timer := &noopTimer{}
	ctx := NewContext(context.Background(), timer)
	assert.Equal(s.T(), timer, fromContext(ctx))

	ctx1, c1 := NewParent(context.Background(), "SomeName")
	assert.NotNil(s.T(), ctx1)
	assert.NotNil(s.T(), c1)
	assert.Equal(s.T(), defaultTimer, fromContext(ctx1))

	c2 := NewChild(context.Background(), "SomeName")
	assert.NotNil(s.T(), c2)
}

// TestTimer validates that we're reporting the correct timing metrics
// by validating the log messages emitted by NewRelic
func (s *MetricTestSuite) TestTimer() {

	txnName := "Txn"
	// Wrap into a function so we can leverage defer
	func() {
		ctx, closeTxn := s.timer.new(context.Background(), txnName)
		assert.NotNil(s.T(), ctx)
		assert.NotNil(s.T(), closeTxn)
		defer closeTxn()

		close1 := s.timer.newChild(ctx, "child")
		assert.NotNil(s.T(), close1)
		defer close1()

	}()

	entries := s.hook.AllEntries()
	// Unfortunately, we do not receive a log entry for closing a segment.
	assert.Equal(s.T(), 1, len(entries))
	assert.NotNil(s.T(), entries[0].Data["duration_ms"])
	assert.Contains(s.T(), entries[0].Data["name"], txnName)
}

func (s *MetricTestSuite) TestTimerNoParent() {
	close := s.timer.newChild(context.Background(), "someChild")
	assert.NotNil(s.T(), close)
	close()

	entries := s.hook.AllEntries()
	assert.Equal(s.T(), 1, len(entries))
	assert.Contains(s.T(), entries[0].Message, "No transaction found. Cannot create child.")
}

func (s *MetricTestSuite) TestNoOpTimer() {
	timer := &noopTimer{}
	ctx, closeTxn := timer.new(context.Background(), "someTxnName")
	assert.NotNil(s.T(), ctx)
	assert.NotNil(s.T(), closeTxn)
	assert.Equal(s.T(), context.Background(), ctx)

	closeChild := timer.newChild(ctx, "someChildName")
	assert.NotNil(s.T(), closeChild)
}

// TestDefaultTimer validates that we return a non-nil timer
// when we cannot instantiate a NewRelic backed one.
func (s *MetricTestSuite) TestDefaultTimer() {
	t := GetTimer()
	assert.NotNil(s.T(), t)
	assert.IsType(s.T(), &noopTimer{}, t)
}
