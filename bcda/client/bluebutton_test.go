package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BBTestSuite struct {
	suite.Suite
	bbClient *BlueButtonClient
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

	if bbClient, err := NewBlueButtonClient(map[string]string{jobIDKey: "543210", cmsIDKey: "A00234"}); err != nil {
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
	bbc, err := NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_CERT_FILE", "foo.pem")
	bbc, err = NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCertFile() {
	origCertFile := os.Getenv("BB_CLIENT_CERT_FILE")
	defer os.Setenv("BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_CERT_FILE", "../static/emptyFile.pem")
	bbc, err := NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_CERT_FILE", "../static/badPublic.pem")
	bbc, err = NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoKeyFile() {
	origKeyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	defer os.Setenv("BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	os.Unsetenv("BB_CLIENT_KEY_FILE")
	bbc, err := NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_KEY_FILE", "foo.pem")
	bbc, err = NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidKeyFile() {
	origKeyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	defer os.Setenv("BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_KEY_FILE", "../static/emptyFile.pem")
	bbc, err := NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not load Blue Button keypair")

	os.Setenv("BB_CLIENT_KEY_FILE", "../static/badPublic.pem")
	bbc, err = NewBlueButtonClient(map[string]string{})
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
	bbc, err := NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read CA file")

	os.Setenv("BB_CLIENT_CA_FILE", "foo.pem")
	bbc, err = NewBlueButtonClient(map[string]string{})
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
	bbc, err := NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.EqualError(err, "could not append CA certificate(s)")

	os.Setenv("BB_CLIENT_CA_FILE", "../static/badPublic.pem")
	bbc, err = NewBlueButtonClient(map[string]string{})
	assert.Nil(bbc)
	assert.NotNil(err)
	assert.EqualError(err, "could not append CA certificate(s)")
}

func (s *BBTestSuite) TestGetBlueButtonPatientData() {
	p, err := s.bbClient.GetPatientData("012345")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), p, `{ "test": "ok"`)
	assert.NotContains(s.T(), p, "excludeSAMHSA=true")
}

func (s *BBTestSuite) TestGetBlueButtonCoverageData() {
	c, err := s.bbClient.GetCoverageData("012345")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), c, `{ "test": "ok"`)
	assert.NotContains(s.T(), c, "excludeSAMHSA=true")
}

func (s *BBTestSuite) TestGetBlueButtonExplanationOfBenefitData() {
	e, err := s.bbClient.GetExplanationOfBenefitData("012345")
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
	params := GetDefaultParams()
	assert.Equal(s.T(), "application/fhir+json", params.Get("_format"))
	assert.Equal(s.T(), "", params.Get("patient"))
	assert.Equal(s.T(), "", params.Get("beneficiary"))

}

// Sample values from https://confluence.cms.gov/pages/viewpage.action?spaceKey=BB&title=Getting+Started+with+Blue+Button+2.0%27s+Backend#space-menu-link-content
func (s *BBTestSuite) TestHashHICN() {
	HICN := "1000067585"
	HICNHash := HashHICN(HICN)
	assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
	HICN = "123456789"
	HICNHash = HashHICN(HICN)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
}

func (s *BBTestSuite) TestAddRequestHeaders() {

	bbServer := os.Getenv("BB_SERVER_LOCATION")

	req, err := http.NewRequest("GET", bbServer, nil)
	assert.Nil(s.T(), err)
	reqID := uuid.NewRandom()
	assert.Nil(s.T(), err)

	params := url.Values{}
	params.Set("_format", "application/fhir+json")

	req.URL.RawQuery = params.Encode()
	s.bbClient.addRequestHeaders(req, reqID)

	assert.Equal(s.T(), reqID.String(), req.Header.Get("BlueButton-OriginalQueryId"))
	assert.Equal(s.T(), "1", req.Header.Get("BlueButton-OriginalQueryCounter"))
	assert.Equal(s.T(), "", req.Header.Get("BlueButton-BeneficiaryId"))
	assert.Equal(s.T(), "", req.Header.Get("BlueButton-OriginatingIpAddress"))

	assert.Equal(s.T(), "", req.Header.Get("keep-alive"))
	assert.Equal(s.T(), "https", req.Header.Get("X-Forwarded-Proto"))
	assert.Equal(s.T(), "", req.Header.Get("X-Forwarded-Host"))

	assert.Equal(s.T(), req.URL.String(), req.Header.Get("BlueButton-OriginalUrl"))
	assert.Equal(s.T(), req.URL.RawQuery, req.Header.Get("BlueButton-OriginalQuery"))
	assert.Equal(s.T(), "", req.Header.Get("BlueButton-BackendCall"))

	assert.Equal(s.T(), "543210", req.Header.Get(jobIDKey))
	assert.Equal(s.T(), "A00234", req.Header.Get(cmsIDKey))

}

func (s *BBTestSuite) TearDownTest() {
	s.ts.Close()
}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
}
