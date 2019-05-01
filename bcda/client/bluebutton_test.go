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
		w.Header().Set("Content-Type", r.URL.Query().Get("_format"))
		response := fmt.Sprintf("{ \"test\": \"ok\"; \"url\": %v}", r.URL.String())
		fmt.Fprint(w, response)
	}))

	os.Setenv("BB_SERVER_LOCATION", s.ts.URL)
	os.Setenv("BB_CLIENT_CERT_FILE", "../../shared_files/bb-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../../shared_files/bb-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")

	if bbClient, err := client.NewBlueButtonClient(); err != nil {
		s.Fail("Failed to create Blue Button client", err)
	} else {
		s.bbClient = bbClient
	}
}

func (s *BBTestSuite) TestNewBlueButtonClientNoCertFile() {
	origCertFile := os.Getenv("BB_CLIENT_CERT_FILE")
	defer os.Setenv("BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	os.Unsetenv("BB_CLIENT_CERT_FILE")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_CERT_FILE", "foo.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCertFile() {
	origCertFile := os.Getenv("BB_CLIENT_CERT_FILE")
	defer os.Setenv("BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_CERT_FILE", "../static/emptyFile.pem")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_CERT_FILE", "../static/badPublic.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoKeyFile() {
	origKeyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	defer os.Setenv("BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	os.Unsetenv("BB_CLIENT_KEY_FILE")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_KEY_FILE", "foo.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidKeyFile() {
	origKeyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	defer os.Setenv("BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_KEY_FILE", "../static/emptyFile.pem")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_KEY_FILE", "../static/badPublic.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoCAFile() {
	origCAFile := os.Getenv("BB_CLIENT_CA_FILE")
	origCheckCert := os.Getenv("BB_CHECK_CERT")
	defer func() {
		os.Setenv("BB_CLIENT_CA_FILE", origCAFile)
		os.Setenv("BB_CHECK_CERT", origCheckCert)
	}()

	assert := assert.New(s.T())

	os.Unsetenv("BB_CLIENT_CA_FILE")
	os.Unsetenv("BB_CHECK_CERT")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read CA file")

	os.Setenv("BB_CLIENT_CA_FILE", "foo.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read CA file")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCAFile() {
	origCAFile := os.Getenv("BB_CLIENT_CA_FILE")
	origCheckCert := os.Getenv("BB_CHECK_CERT")
	defer func() {
		os.Setenv("BB_CLIENT_CA_FILE", origCAFile)
		os.Setenv("BB_CHECK_CERT", origCheckCert)
	}()

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_CA_FILE", "../static/emptyFile.pem")
	os.Unsetenv("BB_CHECK_CERT")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.EqualError(err, "could not append CA certificate(s)")

	os.Setenv("BB_CLIENT_CA_FILE", "../static/badPublic.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.EqualError(err, "could not append CA certificate(s)")
}

func (s *BBTestSuite) TestGetBlueButtonPatientData() {
	p, err := s.bbClient.GetPatientData("012345", "543210")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), p, `{ "test": "ok"`)
	assert.NotContains(s.T(), p, "excludeSAMHSA=true")
}

func (s *BBTestSuite) TestGetBlueButtonCoverageData() {
	c, err := s.bbClient.GetCoverageData("012345", "543210")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), c, `{ "test": "ok"`)
	assert.NotContains(s.T(), c, "excludeSAMHSA=true")
}

func (s *BBTestSuite) TestGetBlueButtonExplanationOfBenefitData() {
	e, err := s.bbClient.GetExplanationOfBenefitData("012345", "543210")
	assert.Nil(s.T(), err)

	assert.Contains(s.T(), e, `{ "test": "ok"`)
	assert.Contains(s.T(), e, "excludeSAMHSA=true")
}

func (s *BBTestSuite) TestGetBlueButtonMetadata() {
	m, err := s.bbClient.GetMetadata()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), m, `{ "test": "ok"`)
	assert.NotContains(s.T(), m, "excludeSAMHSA=true")
}

func (s *BBTestSuite) TestGetDefaultParams() {
	params := client.GetDefaultParams()
	assert.Equal(s.T(), "application/fhir+json", params.Get("_format"))
	assert.Equal(s.T(), "", params.Get("patient"))
	assert.Equal(s.T(), "", params.Get("beneficiary"))

}

// Sample values from https://confluence.cms.gov/pages/viewpage.action?spaceKey=BB&title=Getting+Started+with+Blue+Button+2.0%27s+Backend#space-menu-link-content
func (s *BBTestSuite) TestHashHICN() {
	HICN := "1000067585"
	HICNHash := client.HashHICN(HICN)
	assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
	HICN = "123456789"
	HICNHash = client.HashHICN(HICN)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
}

func (s *BBTestSuite) TearDownTest() {
	s.ts.Close()
}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
}
