package client_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BBTestSuite struct {
	suite.Suite
}

type BBRequestTestSuite struct {
	BBTestSuite
	bbClient *client.BlueButtonClient
	ts       *httptest.Server
}

var ts200, ts500 *httptest.Server

func (s *BBTestSuite) SetupSuite() {
	os.Setenv("BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	os.Setenv("BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	os.Setenv("BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")
}

func (s *BBRequestTestSuite) SetupSuite() {
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

func (s *BBRequestTestSuite) BeforeTest(suiteName, testName string) {
	if strings.Contains(testName, "500") {
		s.ts = ts500
	} else {
		s.ts = ts200
	}
	os.Setenv("BB_SERVER_LOCATION", s.ts.URL)
}

/* Tests for creating client and other functions that don't make requests */
func (s *BBTestSuite) TestNewBlueButtonClientNoCertFile() {
	origCertFile := os.Getenv("BB_CLIENT_CERT_FILE")
	defer os.Setenv("BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	os.Unsetenv("BB_CLIENT_CERT_FILE")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open : no such file or directory")

	os.Setenv("BB_CLIENT_CERT_FILE", "foo.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCertFile() {
	origCertFile := os.Getenv("BB_CLIENT_CERT_FILE")
	defer os.Setenv("BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_CERT_FILE", "../static/emptyFile.pem")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in certificate input")

	os.Setenv("BB_CLIENT_CERT_FILE", "../static/badPublic.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in certificate input")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoKeyFile() {
	origKeyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	defer os.Setenv("BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	os.Unsetenv("BB_CLIENT_KEY_FILE")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open : no such file or directory")

	os.Setenv("BB_CLIENT_KEY_FILE", "foo.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidKeyFile() {
	origKeyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	defer os.Setenv("BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	os.Setenv("BB_CLIENT_KEY_FILE", "../static/emptyFile.pem")
	bbc, err := client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in key input")

	os.Setenv("BB_CLIENT_KEY_FILE", "../static/badPublic.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in key input")
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
	assert.EqualError(err, "could not read CA file: read .: is a directory")

	os.Setenv("BB_CLIENT_CA_FILE", "foo.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not read CA file: open foo.pem: no such file or directory")
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
	assert.EqualError(err, "could not append CA certificate(s)")

	os.Setenv("BB_CLIENT_CA_FILE", "../static/badPublic.pem")
	bbc, err = client.NewBlueButtonClient()
	assert.Nil(bbc)
	assert.EqualError(err, "could not append CA certificate(s)")
}

func (s *BBTestSuite) TestGetDefaultParams() {
	params := client.GetDefaultParams()
	assert.Equal(s.T(), "application/fhir+json", params.Get("_format"))
	assert.Equal(s.T(), "", params.Get("patient"))
	assert.Equal(s.T(), "", params.Get("beneficiary"))

}

/* Tests that make requests, using clients configured with the 200 response and 500 response httptest.Servers initialized in SetupSuite() */
func (s *BBRequestTestSuite) TestGetPatientWithoutSince() {
	since := ""
	p, err := s.bbClient.GetPatient("012345", "543210", "A0000", since)
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), p, `{ "test": "ok"`)
	assert.NotContains(s.T(), p, "excludeSAMHSA=true")
	assert.NotContains(s.T(), p, "_lastUpdated")
}

func (s *BBRequestTestSuite) TestGetPatientWithInvalidSince_500() {
        since := "invalid"
        p, err := s.bbClient.GetPatient("012345", "543210", "A0000", since)
        assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
        assert.Equal(s.T(), "", p)
}

func (s *BBRequestTestSuite) TestGetPatientWithSince() {
        since := "gt2020-02-14"
        p, err := s.bbClient.GetPatient("012345", "543210", "A0000", since)
        assert.Nil(s.T(), err)
        assert.Contains(s.T(), p, `{ "test": "ok"`)
        assert.NotContains(s.T(), p, "excludeSAMHSA=true")
        assert.Contains(s.T(), p, fmt.Sprintf("_lastUpdated=%s", since))
}

func (s *BBRequestTestSuite) TestGetPatient_500() {
        since := ""
	p, err := s.bbClient.GetPatient("012345", "543210", "A0000", since)
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBRequestTestSuite) TestGetCoverageWithoutSince() {
        since := ""
	c, err := s.bbClient.GetCoverage("012345", "543210", "A0000", since)
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), c, `{ "test": "ok"`)
	assert.NotContains(s.T(), c, "excludeSAMHSA=true")
        assert.NotContains(s.T(), c, "_lastUpdated")
}

func (s *BBRequestTestSuite) TestGetCoverageWithInvalidSince_500() {
        since := "invalid"
        c, err := s.bbClient.GetCoverage("012345", "543210", "A0000", since)
        assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
        assert.Equal(s.T(), "", c)
}

func (s *BBRequestTestSuite) TestGetCoverageWithSince() {
        since := "gt2020-02-14"
        c, err := s.bbClient.GetCoverage("012345", "543210", "A0000", since)
        assert.Nil(s.T(), err)
        assert.Contains(s.T(), c, `{ "test": "ok"`)
        assert.NotContains(s.T(), c, "excludeSAMHSA=true")
        assert.Contains(s.T(), c, fmt.Sprintf("_lastUpdated=%s", since))
}

func (s *BBRequestTestSuite) TestGetCoverage_500() {
        since := ""
	p, err := s.bbClient.GetCoverage("012345", "543210", "A0000", since)
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefitWithoutSince() {
        since := ""
	e, err := s.bbClient.GetExplanationOfBenefit("012345", "543210", "A0000", since)
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), e, `{ "test": "ok"`)
	assert.Contains(s.T(), e, "excludeSAMHSA=true")
        assert.NotContains(s.T(), e, "_lastUpdated")
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefitWithInvalidSince_500() {
        since := "invalid"
        e, err := s.bbClient.GetExplanationOfBenefit("012345", "543210", "A0000", since)
        assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
        assert.Equal(s.T(), "", e)
        assert.NotContains(s.T(), e, "_lastUpdated")
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefitWithSince() {
        since := "gt2020-02-14"
        e, err := s.bbClient.GetExplanationOfBenefit("012345", "543210", "A0000", since)
        assert.Nil(s.T(), err)
        assert.Contains(s.T(), e, `{ "test": "ok"`)
        assert.Contains(s.T(), e, "excludeSAMHSA=true")
        assert.Contains(s.T(), e, fmt.Sprintf("_lastUpdated=%s", since))
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefit_500() {
        since := ""
	p, err := s.bbClient.GetExplanationOfBenefit("012345", "543210", "A0000", since)
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
        assert.NotContains(s.T(), p, "_lastUpdated")
}

func (s *BBRequestTestSuite) TestGetMetadata() {
	m, err := s.bbClient.GetMetadata()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), m, `{ "test": "ok"`)
	assert.NotContains(s.T(), m, "excludeSAMHSA=true")
}

func (s *BBRequestTestSuite) TestGetMetadata_500() {
	p, err := s.bbClient.GetMetadata()
	assert.Regexp(s.T(), `Blue Button request .+ failed \d+ time\(s\)`, err.Error())
	assert.Equal(s.T(), "", p)
}

// Sample values from https://confluence.cms.gov/pages/viewpage.action?spaceKey=BB&title=Getting+Started+with+Blue+Button+2.0%27s+Backend#space-menu-link-content
func (s *BBTestSuite) TestHashIdentifier() {
	assert.NotZero(s.T(), os.Getenv("BB_HASH_PEPPER"))
	HICN := "1000067585"
	HICNHash := client.HashIdentifier(HICN)
	// This test will only be valid for this pepper.  If it is different in different environments we will need different checks
	if os.Getenv("BB_HASH_PEPPER") == "b8ebdcc47fdd852b8b0201835c6273a9177806e84f2d9dc4f7ecaff08681e86d74195c6aef2db06d3d44c9d0b8f93c3e6c43d90724b605ac12585b9ab5ee9c3f00d5c0d284e6b8e49d502415c601c28930637b58fdca72476e31c22ad0f24ecd761020d6a4bcd471f0db421d21983c0def1b66a49a230f85f93097e9a9a8e0a4f4f0add775213cbf9ecfc1a6024cb021bd1ed5f4981a4498f294cca51d3939dfd9e6a1045350ddde7b6d791b4d3b884ee890d4c401ef97b46d1e57d40efe5737248dd0c4cec29c23c787231c4346cab9bb973f140a32abaa0a2bd5c0b91162f8d2a7c9d3347aafc76adbbd90ec5bfe617a3584e94bc31047e3bb6850477219a9" {
		assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
	}
	HICN = "123456789"
	HICNHash = client.HashIdentifier(HICN)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)

	MBI := "1000067585"
	MBIHash := client.HashIdentifier(MBI)
	if os.Getenv("BB_HASH_PEPPER") == "b8ebdcc47fdd852b8b0201835c6273a9177806e84f2d9dc4f7ecaff08681e86d74195c6aef2db06d3d44c9d0b8f93c3e6c43d90724b605ac12585b9ab5ee9c3f00d5c0d284e6b8e49d502415c601c28930637b58fdca72476e31c22ad0f24ecd761020d6a4bcd471f0db421d21983c0def1b66a49a230f85f93097e9a9a8e0a4f4f0add775213cbf9ecfc1a6024cb021bd1ed5f4981a4498f294cca51d3939dfd9e6a1045350ddde7b6d791b4d3b884ee890d4c401ef97b46d1e57d40efe5737248dd0c4cec29c23c787231c4346cab9bb973f140a32abaa0a2bd5c0b91162f8d2a7c9d3347aafc76adbbd90ec5bfe617a3584e94bc31047e3bb6850477219a9" {
		assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", MBIHash)
	}
	MBI = "123456789"
	MBIHash = client.HashIdentifier(MBI)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", MBIHash)
}

func (s *BBRequestTestSuite) TearDownAllSuite() {
	s.ts.Close()
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
	client.AddRequestHeaders(req, reqID, "543210", "A00234")

	assert.Equal(s.T(), reqID.String(), req.Header.Get("BlueButton-OriginalQueryId"))
	assert.Equal(s.T(), "1", req.Header.Get("BlueButton-OriginalQueryCounter"))

	assert.Equal(s.T(), "", req.Header.Get("keep-alive"))
	assert.Equal(s.T(), "https", req.Header.Get("X-Forwarded-Proto"))
	assert.Equal(s.T(), "", req.Header.Get("X-Forwarded-Host"))

	assert.Equal(s.T(), req.URL.String(), req.Header.Get("BlueButton-OriginalUrl"))
	assert.Equal(s.T(), req.URL.RawQuery, req.Header.Get("BlueButton-OriginalQuery"))

	assert.Equal(s.T(), "543210", req.Header.Get("BCDA-JOBID"))
	assert.Equal(s.T(), "A00234", req.Header.Get("BCDA-CMSID"))
	assert.Equal(s.T(), "mbi", req.Header.Get("IncludeIdentifiers"))

}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
	suite.Run(t, new(BBRequestTestSuite))
}
