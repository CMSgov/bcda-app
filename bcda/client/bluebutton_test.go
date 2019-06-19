package client_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BBTestSuite struct {
	suite.Suite
}

type BBMockTestSuite struct {
	BBTestSuite
	bbClient *client.BlueButtonClient
	ts       *httptest.Server
}

var ts200, ts500 *httptest.Server

func (s *BBTestSuite) SetupSuite() {
	os.Setenv("BB_CLIENT_CERT_FILE", "../../shared_files/bb-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../../shared_files/bb-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")
}

func (s *BBMockTestSuite) SetupSuite() {
	ts200 = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", r.URL.Query().Get("_format"))
		response := fmt.Sprintf("{ \"test\": \"ok\"; \"url\": %v}", r.URL.String())
		fmt.Fprint(w, response)
	}))

	ts500 = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Print("")
	}))

	if bbClient, err := client.NewBlueButtonClient(); err != nil {
		s.Fail("Failed to create Blue Button client", err)
	} else {
		s.bbClient = bbClient
	}
}

func (s *BBMockTestSuite) BeforeTest(suiteName, testName string) {
	if strings.Contains(testName, "500") {
		s.ts = ts500
	} else {
		s.ts = ts200
	}
	os.Setenv("BB_SERVER_LOCATION", s.ts.URL)
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

func (s *BBTestSuite) TestGetDefaultParams() {
	params := client.GetDefaultParams()
	assert.Equal(s.T(), "application/fhir+json", params.Get("_format"))
	assert.Equal(s.T(), "", params.Get("patient"))
	assert.Equal(s.T(), "", params.Get("beneficiary"))

}

/* Tests below use mock Blue Button client */
func (s *BBMockTestSuite) TestGetBlueButtonPatientData() {
	p, err := s.bbClient.GetPatientData("012345", "543210")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), p, `{ "test": "ok"`)
	assert.NotContains(s.T(), p, "excludeSAMHSA=true")
}

func (s *BBMockTestSuite) TestGetBlueButtonPatientData_500() {
	p, err := s.bbClient.GetPatientData("012345", "543210")
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBMockTestSuite) TestGetBlueButtonCoverageData() {
	c, err := s.bbClient.GetCoverageData("012345", "543210")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), c, `{ "test": "ok"`)
	assert.NotContains(s.T(), c, "excludeSAMHSA=true")
}

func (s *BBMockTestSuite) TestGetBlueButtonCoverageData_500() {
	p, err := s.bbClient.GetCoverageData("012345", "543210")
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBMockTestSuite) TestGetBlueButtonExplanationOfBenefitData() {
	e, err := s.bbClient.GetExplanationOfBenefitData("012345", "543210")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), e, `{ "test": "ok"`)
	assert.Contains(s.T(), e, "excludeSAMHSA=true")
}

func (s *BBMockTestSuite) TestGetBlueButtonExplanationOfBenefitData_500() {
	p, err := s.bbClient.GetExplanationOfBenefitData("012345", "543210")
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBMockTestSuite) TestGetBlueButtonMetadata() {
	m, err := s.bbClient.GetMetadata()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), m, `{ "test": "ok"`)
	assert.NotContains(s.T(), m, "excludeSAMHSA=true")
}

func (s *BBMockTestSuite) TestGetBlueButtonMetadata_500() {
	p, err := s.bbClient.GetMetadata()
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

// Sample values from https://confluence.cms.gov/pages/viewpage.action?spaceKey=BB&title=Getting+Started+with+Blue+Button+2.0%27s+Backend#space-menu-link-content
func (s *BBTestSuite) TestHashHICN() {
	assert.NotZero(s.T(), os.Getenv("BB_HASH_PEPPER"))
	HICN := "1000067585"
	HICNHash := client.HashHICN(HICN)
	// This test will only be valid for this pepper.  If it is different in different environments we will need different checks
	if os.Getenv("BB_HASH_PEPPER") == "b8ebdcc47fdd852b8b0201835c6273a9177806e84f2d9dc4f7ecaff08681e86d74195c6aef2db06d3d44c9d0b8f93c3e6c43d90724b605ac12585b9ab5ee9c3f00d5c0d284e6b8e49d502415c601c28930637b58fdca72476e31c22ad0f24ecd761020d6a4bcd471f0db421d21983c0def1b66a49a230f85f93097e9a9a8e0a4f4f0add775213cbf9ecfc1a6024cb021bd1ed5f4981a4498f294cca51d3939dfd9e6a1045350ddde7b6d791b4d3b884ee890d4c401ef97b46d1e57d40efe5737248dd0c4cec29c23c787231c4346cab9bb973f140a32abaa0a2bd5c0b91162f8d2a7c9d3347aafc76adbbd90ec5bfe617a3584e94bc31047e3bb6850477219a9" {
		assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
	}
	HICN = "123456789"
	HICNHash = client.HashHICN(HICN)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
}

func (s *BBMockTestSuite) TearDownAllSuite() {
	s.ts.Close()
}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
	suite.Run(t, new(BBMockTestSuite))
}
