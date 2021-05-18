package insights

import (
	"testing"

	"github.com/CMSgov/bcda-app/conf"

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

func InsightsTestSuite(t *testing.T) {
	suite.Run(t, new(InsightsTestSuite))
}

func PutRecord(*firehose.PutRecordInput) (*firehose.PutRecordOutput, error) {
	// TODO
}

func (s *InsightsTestSuite) TestInsightsDisabled() {
	origSetting := conf.GetEnv("BCDA_ENABLE_INSIGHTS_EVENTS")
	conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", false)

	s.T().Cleanup(func() {
		conf.SetEnv(s.T(), "BCDA_ENABLE_INSIGHTS_EVENTS", origSetting)
	})

	PutEvent(s.mockSvc, "TestInsightsDisabledName", "TestInsightsDisabledEvent")

	// TODO: assert
}
