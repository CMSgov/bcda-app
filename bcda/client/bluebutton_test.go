package client_test

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/twinj/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	clientIDHeader    = "BULK-CLIENTID"
	jobIDHeader       = "BULK-JOBID"
	oldClientIDHeader = "BCDA-JOBID"
	oldJobIDHeader    = "BCDA-CMSID"
)

type BBTestSuite struct {
	suite.Suite
}

type BBRequestTestSuite struct {
	BBTestSuite
	bbClient *client.BlueButtonClient
	ts       *httptest.Server
}

var (
	ts200, ts500 *httptest.Server
	logger       = testUtils.GetLogger(logrus.StandardLogger())
	now          = time.Now()
	nowFormatted = url.QueryEscape(now.Format(time.RFC3339Nano))
	since        = "gt2020-02-14"
	claimsDate   = client.ClaimsWindow{LowerBound: time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC),
		UpperBound: time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)}
	jobData = models.JobEnqueueArgs{ID: 1, CMSID: "A0000", Since: since, TransactionTime: now}
)

func (s *BBTestSuite) SetupSuite() {
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")
	conf.SetEnv(s.T(), "BB_REQUEST_RETRY_INTERVAL_MS", "10")
	conf.SetEnv(s.T(), "BB_TIMEOUT_MS", "2000")

	// Set up the logger since we're using the real client
	client.SetLogger(logger)
}

func (s *BBRequestTestSuite) SetupSuite() {
	ts200 = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerFunc(w, r, false)
	}))

	ts500 = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Some server error", http.StatusInternalServerError)
	}))
}

func (s *BBRequestTestSuite) BeforeTest(suiteName, testName string) {
	client.SetLogger(logger)
	if strings.Contains(testName, "500") {
		s.ts = ts500
	} else {
		s.ts = ts200
	}

	config := client.BlueButtonConfig{
		BBServer: s.ts.URL,
	}
	if bbClient, err := client.NewBlueButtonClient(config); err != nil {
		s.Fail("Failed to create Blue Button client", err)
	} else {
		s.bbClient = bbClient
	}
}

/* Tests for creating client and other functions that don't make requests */
func (s *BBTestSuite) TestNewBlueButtonClientNoCertFile() {
	origCertFile := conf.GetEnv("BB_CLIENT_CERT_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	conf.UnsetEnv(s.T(), "BB_CLIENT_CERT_FILE")
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open : no such file or directory")

	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", constants.TestKeyName)
	bbc, err = client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCertFile() {
	origCertFile := conf.GetEnv("BB_CLIENT_CERT_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", constants.EmptyKeyName)
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in certificate input")

	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", constants.BadKeyName)
	bbc, err = client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in certificate input")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoKeyFile() {
	origKeyFile := conf.GetEnv("BB_CLIENT_KEY_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	conf.UnsetEnv(s.T(), "BB_CLIENT_KEY_FILE")
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open : no such file or directory")

	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", constants.TestKeyName)
	bbc, err = client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidKeyFile() {
	origKeyFile := conf.GetEnv("BB_CLIENT_KEY_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", constants.EmptyKeyName)
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in key input")

	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", constants.BadKeyName)
	bbc, err = client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in key input")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoCAFile() {
	origCAFile := conf.GetEnv("BB_CLIENT_CA_FILE")
	origCheckCert := conf.GetEnv("BB_CHECK_CERT")
	defer func() {
		conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", origCAFile)
		conf.SetEnv(s.T(), "BB_CHECK_CERT", origCheckCert)
	}()

	assert := assert.New(s.T())

	conf.UnsetEnv(s.T(), "BB_CLIENT_CA_FILE")
	conf.UnsetEnv(s.T(), "BB_CHECK_CERT")
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not read CA file: read .: is a directory")

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", constants.TestKeyName)
	bbc, err = client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not read CA file: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCAFile() {
	origCAFile := conf.GetEnv("BB_CLIENT_CA_FILE")
	origCheckCert := conf.GetEnv("BB_CHECK_CERT")
	defer func() {
		conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", origCAFile)
		conf.SetEnv(s.T(), "BB_CHECK_CERT", origCheckCert)
	}()

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", constants.EmptyKeyName)
	conf.UnsetEnv(s.T(), "BB_CHECK_CERT")
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not append CA certificate(s)")

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", constants.BadKeyName)
	bbc, err = client.NewBlueButtonClient(client.NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not append CA certificate(s)")
}

func (s *BBTestSuite) TestNewBlueButtonClientMultipleCaFiles() {
	origCertFile := conf.GetEnv("BB_CLIENT_CA_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", origCertFile)

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt,../../shared_files/localhost.crt")
	bbc, err := client.NewBlueButtonClient(client.NewConfig(""))
	assert.NotNil(bbc)
	assert.Nil(err)
}

func (s *BBTestSuite) TestGetDefaultParams() {
	params := client.GetDefaultParams()
	assert.Equal(s.T(), "application/fhir+json", params.Get("_format"))
	assert.Equal(s.T(), "", params.Get("patient"))
	assert.Equal(s.T(), "", params.Get("beneficiary"))

}

func (s *BBRequestTestSuite) TestGetBBLogs() {
	hook := test.NewLocal(logger)
	_, err := s.bbClient.GetPatient(jobData, "012345")
	var logCMSID, logJobID bool
	for _, entry := range hook.AllEntries() {
		test := entry.Data
		s.T().Log(test)
		if entry.Data["cms_id"] == jobData.CMSID {
			logCMSID = true
		}
		if entry.Data["job_id"] == strconv.Itoa(jobData.ID) {
			logJobID = true
		}
	}
	assert.True(s.T(), logCMSID)
	assert.True(s.T(), logJobID)
	assert.Nil(s.T(), err)
}

/* Tests that make requests, using clients configured with the 200 response and 500 response httptest.Servers initialized in SetupSuite() */
func (s *BBRequestTestSuite) TestGetPatient() {
	p, err := s.bbClient.GetPatient(jobData, "012345")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(p.Entries))
	assert.Equal(s.T(), "20000000000001", p.Entries[0]["resource"].(map[string]interface{})["id"])
}

func (s *BBRequestTestSuite) TestGetPatient_500() {
	p, err := s.bbClient.GetPatient(jobData, "012345")
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), p)
}

func (s *BBRequestTestSuite) TestGetCoverage() {
	c, err := s.bbClient.GetCoverage(jobData, "012345")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 3, len(c.Entries))
	assert.Equal(s.T(), "part-b-20000000000001", c.Entries[1]["resource"].(map[string]interface{})["id"])
}

func (s *BBRequestTestSuite) TestGetCoverage_500() {
	c, err := s.bbClient.GetCoverage(jobData, "012345")
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), c)
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefit() {
	e, err := s.bbClient.GetExplanationOfBenefit(jobData, "012345", client.ClaimsWindow{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 33, len(e.Entries))
	assert.Equal(s.T(), "carrier-10525061996", e.Entries[3]["resource"].(map[string]interface{})["id"])
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefit_500() {
	e, err := s.bbClient.GetExplanationOfBenefit(jobData, "012345", client.ClaimsWindow{})
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), e)
}

func (s *BBRequestTestSuite) TestGetClaim() {
	e, err := s.bbClient.GetClaim(jobData, "1234567890hashed", client.ClaimsWindow{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(e.Entries))
}

func (s *BBRequestTestSuite) TestGetClaim_HashIdentifierError() {
	existingPepper := conf.GetEnv("BB_HASH_PEPPER")

	defer func() {
		conf.SetEnv(s.T(), "BB_HASH_PEPPER", existingPepper)
	}()

	conf.SetEnv(s.T(), "BB_HASH_PEPPER", "Ã«ÃÃ¬Ã¹Ã")

	_, err := s.bbClient.GetClaim(jobData, "1234567890hashed", client.ClaimsWindow{})
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "Failed to decode bluebutton hash pepper")
}

func (s *BBRequestTestSuite) TestGetClaim_500() {
	e, err := s.bbClient.GetClaim(jobData, "1234567890hashed", client.ClaimsWindow{})
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), e)
}

func (s *BBRequestTestSuite) TestGetClaimResponse() {
	e, err := s.bbClient.GetClaimResponse(jobData, "1234567890hashed", client.ClaimsWindow{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(e.Entries))
}

func (s *BBRequestTestSuite) TestGetClaimResponse_HashIdentifierError() {
	existingPepper := conf.GetEnv("BB_HASH_PEPPER")

	defer func() {
		conf.SetEnv(s.T(), "BB_HASH_PEPPER", existingPepper)
	}()

	conf.SetEnv(s.T(), "BB_HASH_PEPPER", "Ã«ÃÃ¬Ã¹Ã")

	_, err := s.bbClient.GetClaimResponse(jobData, "1234567890hashed", client.ClaimsWindow{})
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "Failed to decode bluebutton hash pepper")
}

func (s *BBRequestTestSuite) TestGetClaimResponse_500() {
	e, err := s.bbClient.GetClaimResponse(jobData, "1234567890hashed", client.ClaimsWindow{})
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), e)
}

func (s *BBRequestTestSuite) TestGetMetadata() {
	m, err := s.bbClient.GetMetadata()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), m, `"resourceType": "CapabilityStatement"`)
	assert.NotContains(s.T(), m, constants.TestExcludeSAMHSA)
}

func (s *BBRequestTestSuite) TestGetMetadata_500() {
	p, err := s.bbClient.GetMetadata()
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get response`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBRequestTestSuite) TestGetPatientByIdentifierHash() {
	p, err := s.bbClient.GetPatientByIdentifierHash(models.JobEnqueueArgs{}, "hashedIdentifier")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), p, `"id": "20000000000001"`)
}

func (s *BBRequestTestSuite) TestGetPatientByIdentifierHash_500() {
	var cms_id, job_id bool
	hook := test.NewLocal(logrus.StandardLogger())
	jobData := models.JobEnqueueArgs{
		ID:    1,
		CMSID: "A0000",
	}
	p, err := s.bbClient.GetPatientByIdentifierHash(jobData, "hashedIdentifier")
	entry := hook.AllEntries()
	for _, t := range entry {
		s.T().Log(t.Data)
		if _, ok := t.Data["cms_id"]; ok {
			cms_id = true
		}
		if _, ok := t.Data["job_id"]; ok {
			job_id = true
		}
	}
	assert.True(s.T(), cms_id, "Log entry should have a value for field `cms_id`.")
	assert.True(s.T(), job_id, "Log entry should have a value for field `job_id`.")
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get response`, err.Error())
	assert.Equal(s.T(), "", p)
}

// Sample values from https://confluence.cms.gov/pages/viewpage.action?spaceKey=BB&title=Getting+Started+with+Blue+Button+2.0%27s+Backend#space-menu-link-content
func (s *BBTestSuite) TestHashIdentifier() {
	assert.NotZero(s.T(), conf.GetEnv("BB_HASH_PEPPER"))
	HICN := "1000067585"
	HICNHash, err := client.HashIdentifier(HICN)
	assert.Nil(s.T(), err)

	// This test will only be valid for this pepper.  If it is different in different environments we will need different checks
	if conf.GetEnv("BB_HASH_PEPPER") == "b8ebdcc47fdd852b8b0201835c6273a9177806e84f2d9dc4f7ecaff08681e86d74195c6aef2db06d3d44c9d0b8f93c3e6c43d90724b605ac12585b9ab5ee9c3f00d5c0d284e6b8e49d502415c601c28930637b58fdca72476e31c22ad0f24ecd761020d6a4bcd471f0db421d21983c0def1b66a49a230f85f93097e9a9a8e0a4f4f0add775213cbf9ecfc1a6024cb021bd1ed5f4981a4498f294cca51d3939dfd9e6a1045350ddde7b6d791b4d3b884ee890d4c401ef97b46d1e57d40efe5737248dd0c4cec29c23c787231c4346cab9bb973f140a32abaa0a2bd5c0b91162f8d2a7c9d3347aafc76adbbd90ec5bfe617a3584e94bc31047e3bb6850477219a9" {
		assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
	}
	HICN = "123456789"
	HICNHash, err = client.HashIdentifier(HICN)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)

	MBI := "1000067585"
	MBIHash, err := client.HashIdentifier(MBI)
	assert.Nil(s.T(), err)

	if conf.GetEnv("BB_HASH_PEPPER") == "b8ebdcc47fdd852b8b0201835c6273a9177806e84f2d9dc4f7ecaff08681e86d74195c6aef2db06d3d44c9d0b8f93c3e6c43d90724b605ac12585b9ab5ee9c3f00d5c0d284e6b8e49d502415c601c28930637b58fdca72476e31c22ad0f24ecd761020d6a4bcd471f0db421d21983c0def1b66a49a230f85f93097e9a9a8e0a4f4f0add775213cbf9ecfc1a6024cb021bd1ed5f4981a4498f294cca51d3939dfd9e6a1045350ddde7b6d791b4d3b884ee890d4c401ef97b46d1e57d40efe5737248dd0c4cec29c23c787231c4346cab9bb973f140a32abaa0a2bd5c0b91162f8d2a7c9d3347aafc76adbbd90ec5bfe617a3584e94bc31047e3bb6850477219a9" {
		assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", MBIHash)
	}

	MBI = "123456789"
	MBIHash, err = client.HashIdentifier(MBI)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", MBIHash)
}

func (s *BBTestSuite) TestHashIdentifierFailure() {
	assert.NotZero(s.T(), conf.GetEnv("BB_HASH_PEPPER"))
	existingPepper := conf.GetEnv("BB_HASH_PEPPER")

	defer func() {
		conf.SetEnv(s.T(), "BB_HASH_PEPPER", existingPepper)
	}()

	conf.SetEnv(s.T(), "BB_HASH_PEPPER", "Ã«ÃÃ¬Ã¹Ã")

	HICN := "1000067585"
	_, err := client.HashIdentifier(HICN)
	assert.NotNil(s.T(), err)
}

func (s *BBRequestTestSuite) TearDownAllSuite() {
	s.ts.Close()
}

func (s *BBRequestTestSuite) TestValidateRequest() {
	old := conf.GetEnv("BB_CLIENT_PAGE_SIZE")
	jobDataNoSince := models.JobEnqueueArgs{ID: 1, CMSID: "A0000", Since: "", TransactionTime: now}
	defer conf.SetEnv(s.T(), "BB_CLIENT_PAGE_SIZE", old)
	conf.SetEnv(s.T(), "BB_CLIENT_PAGE_SIZE", "0") // Need to ensure that requests do not have the _count parameter

	tests := []struct {
		name          string
		funcUnderTest func(bbClient *client.BlueButtonClient) (interface{}, error)
		// Lighter validation checks since we've already thoroughly tested the methods in other tests
		payloadChecker func(t *testing.T, payload interface{})
		pathCheckers   []func(t *testing.T, req *http.Request)
	}{
		{
			"GetExplanationOfBenefit",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", client.ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				excludeSAMHSAChecker,
				noServiceDateChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitNoSince",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobDataNoSince, "patient1", client.ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				noSinceChecker,
				nowChecker,
				excludeSAMHSAChecker,
				noServiceDateChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitWithUpperBoundServiceDate",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", client.ClaimsWindow{UpperBound: claimsDate.UpperBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				excludeSAMHSAChecker,
				serviceDateUpperBoundChecker,
				noServiceDateLowerBoundChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitWithLowerBoundServiceDate",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", client.ClaimsWindow{LowerBound: claimsDate.LowerBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				excludeSAMHSAChecker,
				serviceDateLowerBoundChecker,
				noServiceDateUpperBoundChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitWithLowerAndUpperBoundServiceDate",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", claimsDate)
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				excludeSAMHSAChecker,
				serviceDateLowerBoundChecker,
				serviceDateUpperBoundChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetPatient",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetPatient(jobData, "patient2")
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				noExcludeSAMHSAChecker,
				includeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetPatientNoSince",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetPatient(jobDataNoSince, "patient2")
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				noSinceChecker,
				nowChecker,
				noExcludeSAMHSAChecker,
				includeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetCoverage",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetCoverage(jobData, "beneID1")
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				noExcludeSAMHSAChecker,
				noIncludeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetCoverageNoSince",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetCoverage(jobDataNoSince, "beneID1")
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				noSinceChecker,
				nowChecker,
				noExcludeSAMHSAChecker,
				noIncludeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetPatientByIdentifierHash",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetPatientByIdentifierHash(models.JobEnqueueArgs{}, "hashedIdentifier")
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(string)
				assert.True(t, ok)
				assert.NotEmpty(t, result)
			},
			[]func(*testing.T, *http.Request){
				noExcludeSAMHSAChecker,
				noIncludeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				noBulkRequestHeaders,
				hasDefaultRequestHeaders,
			},
		},
		{
			"GetClaim",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", client.ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				noServiceDateChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimNoSinceChecker",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobDataNoSince, "beneID1", client.ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				noSinceChecker,
				nowChecker,
				noServiceDateChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimNoServiceDateUpperBound",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", client.ClaimsWindow{LowerBound: claimsDate.LowerBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				serviceDateLowerBoundChecker,
				noServiceDateUpperBoundChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimNoServiceDateLowerBound",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", client.ClaimsWindow{UpperBound: claimsDate.UpperBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				noServiceDateLowerBoundChecker,
				serviceDateUpperBoundChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimWithUpperAndLowerBoundServiceDate",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", claimsDate)
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				serviceDateLowerBoundChecker,
				serviceDateUpperBoundChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimResponse",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", client.ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				noServiceDateChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimResponseNoSinceChecker",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobDataNoSince, "beneID1", client.ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				noSinceChecker,
				nowChecker,
				noServiceDateChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimResponseNoServiceDateUpperBound",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", client.ClaimsWindow{LowerBound: claimsDate.LowerBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				serviceDateLowerBoundChecker,
				noServiceDateUpperBoundChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimResponseNoServiceDateLowerBound",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", client.ClaimsWindow{UpperBound: claimsDate.UpperBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				noServiceDateLowerBoundChecker,
				serviceDateUpperBoundChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetClaimResponseWithUpperAndLowerBoundServiceDate",
			func(bbClient *client.BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", claimsDate)
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				sinceChecker,
				nowChecker,
				serviceDateLowerBoundChecker,
				serviceDateUpperBoundChecker,
				excludeSAMHSAChecker,
				includeTaxNumbersChecker,
				noIncludeAddressFieldsChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {

			tsValidation := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				uid, err := uuid.Parse(req.Header.Get("BlueButton-OriginalQueryId"))
				if err != nil {
					assert.FailNow(t, err.Error())
				}
				assert.NotNil(t, uid)
				assert.Equal(t, "1", req.Header.Get("BlueButton-OriginalQueryCounter"))

				assert.Empty(t, req.Header.Get("keep-alive"))
				assert.Nil(t, req.Header.Values("X-Forwarded-Proto"))
				assert.Nil(t, req.Header.Values("X-Forwarded-Host"))

				assert.True(t, strings.HasSuffix(req.Header.Get("BlueButton-OriginalUrl"), req.URL.String()),
					"%s does not end with %s", req.Header.Get("BlueButton-OriginalUrl"), req.URL.String())
				assert.Equal(t, req.URL.RawQuery, req.Header.Get("BlueButton-OriginalQuery"))

				assert.Empty(t, req.Header.Get(oldJobIDHeader))
				assert.Empty(t, req.Header.Get(oldClientIDHeader))

				assert.Equal(t, "mbi", req.Header.Get("IncludeIdentifiers"))

				// Verify that we have compression enabled on these HTTP requests.
				// NOTE: This header should not be explicitly set on the client. It should be added by the http.Transport.
				// Details: https://golang.org/src/net/http/transport.go#L2432
				assert.Equal(t, "gzip", req.Header.Get("Accept-Encoding"))

				for _, checker := range tt.pathCheckers {
					checker(t, req)
				}

				handlerFunc(w, req, true)
			}))
			defer tsValidation.Close()

			config := client.BlueButtonConfig{
				BBServer: tsValidation.URL,
			}
			bbClient, err := client.NewBlueButtonClient(config)
			if err != nil {
				assert.FailNow(t, err.Error())
			}

			data, err := tt.funcUnderTest(bbClient)
			if err != nil {
				assert.FailNow(t, err.Error())
			}

			tt.payloadChecker(t, data)
		})
	}
}

func handlerFunc(w http.ResponseWriter, r *http.Request, useGZIP bool) {
	path := r.URL.Path
	var (
		file *os.File
		err  error
	)
	if strings.Contains(path, "Coverage") {
		file, err = os.Open("../../shared_files/synthetic_beneficiary_data/Coverage")
	} else if strings.Contains(path, "ExplanationOfBenefit") {
		file, err = os.Open("../../shared_files/synthetic_beneficiary_data/ExplanationOfBenefit")
	} else if strings.Contains(path, "metadata") {
		file, err = os.Open("./testdata/Metadata.json")
	} else if strings.Contains(path, "Patient") {
		file, err = os.Open("../../shared_files/synthetic_beneficiary_data/Patient")
	} else if strings.Contains(path, "ClaimResponse") {
		file, err = os.Open("../../shared_files/synthetic_beneficiary_data/ClaimResponse")
	} else if strings.Contains(path, "Claim") {
		file, err = os.Open("../../shared_files/synthetic_beneficiary_data/Claim")
	} else {
		err = fmt.Errorf("Unrecognized path supplied %s", path)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer file.Close()

	w.Header().Set("Content-Type", r.URL.Query().Get("_format"))

	if useGZIP {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		if _, err := io.Copy(gz, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := io.Copy(w, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func noSinceChecker(t *testing.T, req *http.Request) {
	assert.NotContains(t, req.URL.String(), "_lastUpdated=gt")
}
func sinceChecker(t *testing.T, req *http.Request) {
	assert.Contains(t, req.URL.String(), fmt.Sprintf("_lastUpdated=%s", since))
}
func noExcludeSAMHSAChecker(t *testing.T, req *http.Request) {
	assert.NotContains(t, req.URL.String(), constants.TestExcludeSAMHSA)
}
func excludeSAMHSAChecker(t *testing.T, req *http.Request) {
	assert.Contains(t, req.URL.String(), constants.TestExcludeSAMHSA)
}
func nowChecker(t *testing.T, req *http.Request) {
	assert.Contains(t, req.URL.String(), fmt.Sprintf("_lastUpdated=le%s", nowFormatted))
}
func noServiceDateChecker(t *testing.T, req *http.Request) {
	assert.Empty(t, req.URL.Query()[constants.TestSvcDate])
}
func serviceDateUpperBoundChecker(t *testing.T, req *http.Request) {
	// We expect that service date only contains YYYY-MM-DD
	assert.Contains(t, req.URL.Query()[constants.TestSvcDate], fmt.Sprintf("le%s", claimsDate.UpperBound.Format(constants.TestSvcDateResult)))
}
func noServiceDateUpperBoundChecker(t *testing.T, req *http.Request) {
	// We expect that service date only contains YYYY-MM-DD
	assert.NotContains(t, req.URL.Query()[constants.TestSvcDate], fmt.Sprintf("le%s", claimsDate.UpperBound.Format(constants.TestSvcDateResult)))
}
func serviceDateLowerBoundChecker(t *testing.T, req *http.Request) {
	// We expect that service date only contains YYYY-MM-DD
	assert.Contains(t, req.URL.Query()[constants.TestSvcDate], fmt.Sprintf("ge%s", claimsDate.LowerBound.Format(constants.TestSvcDateResult)))
}
func noServiceDateLowerBoundChecker(t *testing.T, req *http.Request) {
	// We expect that service date only contains YYYY-MM-DD
	assert.NotContains(t, req.URL.Query()[constants.TestSvcDate], fmt.Sprintf("ge%s", claimsDate.LowerBound.Format(constants.TestSvcDateResult)))
}
func noIncludeAddressFieldsChecker(t *testing.T, req *http.Request) {
	assert.Empty(t, req.Header.Get("IncludeAddressFields"))
}
func includeAddressFieldsChecker(t *testing.T, req *http.Request) {
	assert.Equal(t, "true", req.Header.Get("IncludeAddressFields"))
}
func noIncludeTaxNumbersChecker(t *testing.T, req *http.Request) {
	assert.Empty(t, req.Header.Get("IncludeTaxNumbers"))
}
func includeTaxNumbersChecker(t *testing.T, req *http.Request) {
	assert.Equal(t, "true", req.Header.Get("IncludeTaxNumbers"))
}
func hasDefaultRequestHeaders(t *testing.T, req *http.Request) {
	assert.NotEmpty(t, req.Header.Get(constants.BBHeaderTS))
	assert.NotEmpty(t, req.Header.Get(constants.BBHeaderOriginURL))
	assert.NotEmpty(t, req.Header.Get(constants.BBHeaderOriginQID))
	assert.NotEmpty(t, req.Header.Get(constants.BBHeaderOriginQ))
	assert.NotEmpty(t, req.Header.Get(constants.BBHeaderOriginQC))
}
func hasBulkRequestHeaders(t *testing.T, req *http.Request) {
	assert.NotEmpty(t, req.Header.Get(jobIDHeader))
	assert.NotEmpty(t, req.Header.Get(clientIDHeader))
}
func noBulkRequestHeaders(t *testing.T, req *http.Request) {
	for k := range req.Header {
		assert.NotEqual(t, k, jobIDHeader)
		assert.NotEqual(t, k, clientIDHeader)
	}
}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
	suite.Run(t, new(BBRequestTestSuite))
}
