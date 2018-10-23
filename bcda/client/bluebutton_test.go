package client_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BBTestSuite struct {
	suite.Suite
	bbClient *client.BlueButtonClient
	ts       *httptest.Server
}

func (s *BBTestSuite) SetupTest() {
	s.ts = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		fmt.Fprint(w, `{ "test": "ok" }`)
	}))

	os.Setenv("BB_SERVER_LOCATION", s.ts.URL)
	os.Setenv("BB_CLIENT_CERT_FILE", "../../shared_files/bb-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../../shared_files/bb-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../../shared_files/test-server-cert.pem")

	if bbClient, err := client.NewBlueButtonClient(); err != nil {
		s.Fail("Failed to create Blue Button client", err)
	} else {
		s.bbClient = bbClient
	}
}

func (s *BBTestSuite) TestGetBlueButtonPatientData() {
	p, err := s.bbClient.GetPatientData("012345")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), `{ "test": "ok" }`, p)
}

func (s *BBTestSuite) TestGetBlueButtonCoverageData() {
	c, err := s.bbClient.GetCoverageData("012345")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), `{ "test": "ok" }`, c)
}

func (s *BBTestSuite) TestGetBlueButtonExplanationOfBenefitData() {
	e, err := s.bbClient.GetExplanationOfBenefitData("012345")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), `{ "test": "ok" }`, e)
}

func (s *BBTestSuite) TestGetBlueButtonMetadata() {
	m, err := s.bbClient.GetMetadata()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), `{ "test": "ok" }`, m)
}

func (s *BBTestSuite) TearDownTest() {
	s.ts.Close()
}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
}
