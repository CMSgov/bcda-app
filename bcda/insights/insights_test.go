package insights

import (
	"bytes"
	"testing"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"

	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type mockFirehoseClient struct {
	firehoseiface.FirehoseAPI
}

type InsightsTestSuite struct {
	suite.Suite
	mockSvc mockFirehoseClient
}

func (s *InsightsTestSuite) SetupTest() {
	s.mockSvc = &mockFirehoseClient{}
}

func TestInsightsTestSuite(t *testing.T) {
	suite.Run(t, new(InsightsTestSuite))
}

func PutRecord(*firehose.PutRecordInput) (*firehose.PutRecordOutput, error) {
	
}

func (s *InsightsTestSuite) TestInsightsDisabled() {
	origSetting := conf.GetEnv("BCDA_ENABLE_INSIGHTS_EVENTS")
	conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", false)

	s.T().Cleanup(func() {
		conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", origSetting)
	})

	buf := new(bytes.Buffer)
	log.API.SetOutput(&buf)

	PutEvent(s.mockSvc, "TestInsightsDisabledName", "TestInsightsDisabledEvent")

	assert.Contains(buf.String(), "Insights is not enabled for the application.  No data was sent to BFD.")
	buf.Reset()	
}
