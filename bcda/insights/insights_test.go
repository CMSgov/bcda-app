package insights

import (
	"bytes"
	"testing"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"

	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type mockFirehoseClient struct {
	firehoseiface.FirehoseAPI
}

type InsightsTestSuite struct {
	suite.Suite
	mockSvc *mockFirehoseClient
}

func (s *InsightsTestSuite) SetupTest() {
	s.mockSvc = &mockFirehoseClient{}
}

func TestInsightsTestSuite(t *testing.T) {
	suite.Run(t, new(InsightsTestSuite))
}

func (m *mockFirehoseClient) PutRecord(input *firehose.PutRecordInput) (*firehose.PutRecordOutput, error) {
	log.API.Infof("Mock called with DeliveryStreamName %s and PutRecordInput: %s", *input.DeliveryStreamName, input.Record.Data)
	return nil, nil
}

func (s *InsightsTestSuite) TestInsightsDisabled() {
	origSetting := conf.GetEnv("BCDA_ENABLE_INSIGHTS_EVENTS")
	conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", "false")
	originalLog := log.API

	s.T().Cleanup(func() {
		conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", origSetting)
		log.API = originalLog
	})

	// Override log.API so we can verify the output
	buf := new(bytes.Buffer)
	newLog := logrus.New()
	newLog.SetOutput(buf)
	log.API = newLog

	PutEvent(s.mockSvc, "TestInsightsDisabledName", "TestInsightsDisabledEvent")
	assert.Contains(s.T(), buf.String(), "Insights is not enabled for the application.  No data was sent to BFD.")
}

func (s *InsightsTestSuite) TestInsightsEnabled() {

	origSetting := conf.GetEnv("BCDA_ENABLE_INSIGHTS_EVENTS")
	origEnv := conf.GetEnv("DEPLOYMENT_TARGET")
	conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", "true")
	conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", "unit-test")
	originalLog := log.API

	s.T().Cleanup(func() {
		conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", origSetting)
		conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", origEnv)
		log.API = originalLog
	})

	// Override log.API so we can verify the output
	buf := new(bytes.Buffer)
	newLog := logrus.New()
	newLog.SetOutput(buf)
	log.API = newLog

	PutEvent(s.mockSvc, "TestInsightsEnabledName", "TestInsightsEnabledEvent")
	assert.Contains(s.T(), buf.String(), "TestInsightsEnabledName")
	assert.Contains(s.T(), buf.String(), "TestInsightsEnabledEvent")
	assert.Contains(s.T(), buf.String(), "bfd-insights-bcda-unit-test-TestInsightsEnabledName")
}
