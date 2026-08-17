package client

import (
	"bytes"
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

	"github.com/CMSgov/bcda-app/bcda/client/fhir"
	"github.com/CMSgov/bcda-app/bcda/constants"
	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	oldClientIDHeader = "BCDA-JOBID"
	oldJobIDHeader    = "BCDA-CMSID"
)

type BBTestSuite struct {
	suite.Suite
}

type BBRequestTestSuite struct {
	BBTestSuite
	bbClient *BlueButtonClient
	ts       *httptest.Server
}

var (
	ts200, ts500 *httptest.Server
	now          = time.Now()
	nowFormatted = url.QueryEscape(now.Format(time.RFC3339Nano))
	since        = "gt2020-02-14"
	claimsDate   = ClaimsWindow{LowerBound: time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC),
		UpperBound: time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)}
	jobData = worker_types.JobEnqueueArgs{ID: 1, CMSID: "A0000", Since: since, TransactionID: uuid.New(), TransactionTime: now}
)

func (s *BBTestSuite) SetupSuite() {
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")
	conf.SetEnv(s.T(), "BB_REQUEST_RETRY_INTERVAL_MS", "10")
	conf.SetEnv(s.T(), "BB_TIMEOUT_MS", "2000")

	// Set up the logger since we're using the real client
	logger = testUtils.GetLogger(logrus.StandardLogger())
	SetLogger(logger)
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
	SetLogger(logger)
	if strings.Contains(testName, "500") {
		s.ts = ts500
	} else {
		s.ts = ts200
	}

	config := BlueButtonConfig{
		BBServer: s.ts.URL,
	}
	if bbClient, err := NewBlueButtonClient(config); err != nil {
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
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open : no such file or directory")

	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", constants.TestKeyName)
	bbc, err = NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidCertFile() {
	origCertFile := conf.GetEnv("BB_CLIENT_CERT_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origCertFile)

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", constants.EmptyKeyName)
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in certificate input")

	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", constants.BadKeyName)
	bbc, err = NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in certificate input")
}

func (s *BBTestSuite) TestNewBlueButtonClientNoKeyFile() {
	origKeyFile := conf.GetEnv("BB_CLIENT_KEY_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	conf.UnsetEnv(s.T(), "BB_CLIENT_KEY_FILE")
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open : no such file or directory")

	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", constants.TestKeyName)
	bbc, err = NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: open foo.pem: no such file or directory")
}

func (s *BBTestSuite) TestNewBlueButtonClientInvalidKeyFile() {
	origKeyFile := conf.GetEnv("BB_CLIENT_KEY_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origKeyFile)

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", constants.EmptyKeyName)
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not load Blue Button keypair: tls: failed to find any PEM data in key input")

	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", constants.BadKeyName)
	bbc, err = NewBlueButtonClient(NewConfig(""))
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
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not read CA file: read .: is a directory")

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", constants.TestKeyName)
	bbc, err = NewBlueButtonClient(NewConfig(""))
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
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not append CA certificate(s)")

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", constants.BadKeyName)
	bbc, err = NewBlueButtonClient(NewConfig(""))
	assert.Nil(bbc)
	assert.EqualError(err, "could not append CA certificate(s)")
}

func (s *BBTestSuite) TestNewBlueButtonConfigV3Server() {
	conf.SetEnv(s.T(), "BB_SERVER_LOCATION", "v1-server-location")
	conf.SetEnv(s.T(), "V3_BB_SERVER_LOCATION", "v3-server-location")
	bbc := NewConfig(constants.BFDV1Path)
	assert.Equal(s.T(), bbc.BBServer, "v1-server-location")

	conf.SetEnv(s.T(), "BB_SERVER_LOCATION", "v1-server-location")
	conf.SetEnv(s.T(), "V3_BB_SERVER_LOCATION", "v3-server-location")
	bbc = NewConfig(constants.BFDV3Path)
	assert.Equal(s.T(), bbc.BBServer, "v3-server-location")
}

func (s *BBTestSuite) TestNewBlueButtonClientMultipleCaFiles() {
	origCertFile := conf.GetEnv("BB_CLIENT_CA_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", origCertFile)

	assert := assert.New(s.T())

	conf.SetEnv(s.T(), "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt,../../shared_files/localhost.crt")
	bbc, err := NewBlueButtonClient(NewConfig(""))
	assert.NotNil(bbc)
	assert.Nil(err)
}

func (s *BBTestSuite) TestGetDefaultParams() {
	params := GetDefaultParams()
	assert.Equal(s.T(), "application/fhir+json", params.Get("_format"))
	assert.Equal(s.T(), "", params.Get("patient"))
	assert.Equal(s.T(), "", params.Get("beneficiary"))

}

func (s *BBRequestTestSuite) TestGetBBLogs() {
	hook := test.NewLocal(testUtils.GetLogger(logger))
	_, err := s.bbClient.GetPatient(jobData, "012345")
	var logCMSID, logJobID, logTransID bool
	for _, entry := range hook.AllEntries() {
		test := entry.Data
		s.T().Log(test)
		if entry.Data["cms_id"] == jobData.CMSID {
			logCMSID = true
		}
		if entry.Data["job_id"] == strconv.Itoa(jobData.ID) {
			logJobID = true
		}
		if entry.Data["transaction_id"] == jobData.TransactionID {
			logTransID = true
		}
	}
	assert.True(s.T(), logCMSID)
	assert.True(s.T(), logJobID)
	assert.True(s.T(), logTransID)
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
	e, err := s.bbClient.GetExplanationOfBenefit(jobData, "012345", ClaimsWindow{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 33, len(e.Entries))
	assert.Equal(s.T(), "carrier-10525061996", e.Entries[3]["resource"].(map[string]interface{})["id"])
}

func (s *BBRequestTestSuite) TestGetExplanationOfBenefit_500() {
	e, err := s.bbClient.GetExplanationOfBenefit(jobData, "012345", ClaimsWindow{})
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), e)
}

func (s *BBRequestTestSuite) TestGetClaim() {
	e, err := s.bbClient.GetClaim(jobData, "1234567890hashed", ClaimsWindow{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(e.Entries))
}

func (s *BBRequestTestSuite) TestGetClaim_500() {
	e, err := s.bbClient.GetClaim(jobData, "1234567890hashed", ClaimsWindow{})
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), e)
}

func (s *BBRequestTestSuite) TestGetClaimResponse() {
	e, err := s.bbClient.GetClaimResponse(jobData, "1234567890hashed", ClaimsWindow{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(e.Entries))
}

func (s *BBRequestTestSuite) TestGetClaimResponse_500() {
	e, err := s.bbClient.GetClaimResponse(jobData, "1234567890hashed", ClaimsWindow{})
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get bundle response`, err.Error())
	assert.Nil(s.T(), e)
}

func (s *BBRequestTestSuite) TestGetMetadata() {
	m, err := s.bbClient.GetMetadata()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), m, `"resourceType": "CapabilityStatement"`)
}

func (s *BBRequestTestSuite) TestGetMetadata_500() {
	p, err := s.bbClient.GetMetadata()
	assert.Regexp(s.T(), `blue button request failed \d+ time\(s\) failed to get response`, err.Error())
	assert.Equal(s.T(), "", p)
}

func (s *BBRequestTestSuite) TestGetPatientByMbi() {
	p, err := s.bbClient.GetPatientByMbi(worker_types.JobEnqueueArgs{}, "mbi")
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), p, `"id": "20000000000001"`)
}

func (s *BBRequestTestSuite) TestGetPatientByMbi_500() {
	var cms_id, job_id bool
	hook := test.NewLocal(logrus.StandardLogger())
	jobData := worker_types.JobEnqueueArgs{
		ID:    1,
		CMSID: "A0000",
	}
	p, err := s.bbClient.GetPatientByMbi(jobData, "mbi")
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

func (s *BBRequestTestSuite) TearDownAllSuite() {
	s.ts.Close()
}

func (s *BBRequestTestSuite) TestValidateRequest() {
	old := conf.GetEnv("BB_CLIENT_PAGE_SIZE")
	jobDataNoSince := worker_types.JobEnqueueArgs{ID: 1, CMSID: "A0000", Since: "", TransactionTime: now}
	jobDataWithTypeFilter := worker_types.JobEnqueueArgs{ID: 1, CMSID: "A0000", Since: "gt2020-02-14", TypeFilter: fhir.TypeFilterParameter{
		ResourceType: "ExplanationOfBenefit",
		QueryParameters: []fhir.TypeFilterSubqueryParam{
			{
				Name:  "service-date",
				Value: "gt2022-06-26",
			},
		},
	}, TransactionTime: now}
	defer conf.SetEnv(s.T(), "BB_CLIENT_PAGE_SIZE", old)
	conf.SetEnv(s.T(), "BB_CLIENT_PAGE_SIZE", "0") // Need to ensure that requests do not have the _count parameter

	tests := []struct {
		name          string
		funcUnderTest func(bbClient *BlueButtonClient) (interface{}, error)
		// Lighter validation checks since we've already thoroughly tested the methods in other tests
		payloadChecker func(t *testing.T, payload interface{})
		pathCheckers   []func(t *testing.T, req *http.Request)
	}{
		{
			"GetExplanationOfBenefit",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", ClaimsWindow{})
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
				noSecurityFilterChecker,
				noServiceDateChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitNoSince",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobDataNoSince, "patient1", ClaimsWindow{})
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
				noSecurityFilterChecker,
				noServiceDateChecker,
				noIncludeAddressFieldsChecker,
				includeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitWithUpperBoundServiceDate",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", ClaimsWindow{UpperBound: claimsDate.UpperBound})
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
				noSecurityFilterChecker,
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
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobData, "patient1", ClaimsWindow{LowerBound: claimsDate.LowerBound})
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
				noSecurityFilterChecker,
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
			func(bbClient *BlueButtonClient) (interface{}, error) {
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
				noSecurityFilterChecker,
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
			func(bbClient *BlueButtonClient) (interface{}, error) {
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
				noSecurityFilterChecker,
				includeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetPatientNoSince",
			func(bbClient *BlueButtonClient) (interface{}, error) {
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
				noSecurityFilterChecker,
				includeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetCoverage",
			func(bbClient *BlueButtonClient) (interface{}, error) {
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
				noSecurityFilterChecker,
				noIncludeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetCoverageNoSince",
			func(bbClient *BlueButtonClient) (interface{}, error) {
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
				noSecurityFilterChecker,
				noIncludeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetPatientByMbi",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetPatientByMbi(worker_types.JobEnqueueArgs{}, "mbi")
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(string)
				assert.True(t, ok)
				assert.NotEmpty(t, result)
			},
			[]func(*testing.T, *http.Request){
				noExcludeSAMHSAChecker,
				noSecurityFilterChecker,
				noIncludeAddressFieldsChecker,
				noIncludeTaxNumbersChecker,
				noBulkRequestHeaders,
				hasDefaultRequestHeaders,
				hasContentTypeURLEncodedHeader,
				hasURLEncodedBodyWithIdentifier,
			},
		},
		{
			"GetClaim",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimNoSinceChecker",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobDataNoSince, "beneID1", ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimNoServiceDateUpperBound",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", ClaimsWindow{LowerBound: claimsDate.LowerBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimNoServiceDateLowerBound",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", ClaimsWindow{UpperBound: claimsDate.UpperBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimWithUpperAndLowerBoundServiceDate",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaim(jobData, "beneID1", claimsDate)
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimResponse",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimResponseNoSinceChecker",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobDataNoSince, "beneID1", ClaimsWindow{})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimResponseNoServiceDateUpperBound",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", ClaimsWindow{LowerBound: claimsDate.LowerBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimResponseNoServiceDateLowerBound",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", ClaimsWindow{UpperBound: claimsDate.UpperBound})
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetClaimResponseWithUpperAndLowerBoundServiceDate",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetClaimResponse(jobData, "beneID1", claimsDate)
			},
			func(t *testing.T, payload interface{}) {
				result, ok := payload.(*fhirModels.Bundle)
				assert.True(t, ok)
				assert.NotEmpty(t, result.Entries)
			},
			[]func(*testing.T, *http.Request){
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
				hasClaimRequiredURLEncodedBody,
			},
		},
		{
			"GetExplanationOfBenefitV3",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				bbClient.BBBasePath = constants.BFDV3Path
				return bbClient.GetExplanationOfBenefit(jobDataWithTypeFilter, "patient1", ClaimsWindow{})
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
				securityFilterChecker,
				serviceDateChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitV3_WithClaimsWindow",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				bbClient.BBBasePath = constants.BFDV3Path
				return bbClient.GetExplanationOfBenefit(jobDataWithTypeFilter, "patient1", claimsDate)
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
				securityFilterChecker,
				serviceDateChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
		{
			"GetExplanationOfBenefitWithTypeFilterServiceDate",
			func(bbClient *BlueButtonClient) (interface{}, error) {
				return bbClient.GetExplanationOfBenefit(jobDataWithTypeFilter, "patient1", ClaimsWindow{})
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
				noSecurityFilterChecker,
				serviceDateChecker,
				hasDefaultRequestHeaders,
				hasBulkRequestHeaders,
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {

			tsValidation := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				uid := uuid.Parse(req.Header.Get("BlueButton-OriginalQueryId"))
				assert.NotNil(t, uid)
				assert.Equal(t, "1", req.Header.Get("BlueButton-OriginalQueryCounter"))

				assert.Empty(t, req.Header.Get("keep-alive"))
				assert.Nil(t, req.Header.Values("X-Forwarded-Proto"))
				assert.Nil(t, req.Header.Values("X-Forwarded-Host"))

				assert.True(t, strings.HasSuffix(req.Header.Get("BlueButton-OriginalUrl"), req.URL.String()),
					"%s does not end with %s", req.Header.Get("BlueButton-OriginalUrl"), req.URL.String())

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

			config := BlueButtonConfig{
				BBServer: tsValidation.URL,
			}
			bbClient, err := NewBlueButtonClient(config)
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
func securityFilterChecker(t *testing.T, req *http.Request) {
	assert.Contains(t, req.URL.String(), constants.TestSecurityFilter)
}
func noSecurityFilterChecker(t *testing.T, req *http.Request) {
	assert.NotContains(t, req.URL.String(), constants.TestSecurityFilter)
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
func serviceDateChecker(t *testing.T, req *http.Request) {
	// verify there is max two service-date params set, one upper and one lower
	dates := req.URL.Query()[constants.TestSvcDate]
	assert.True(t, len(dates) <= 2)

	assert.Contains(t, req.URL.String(), "service-date=gt2022-06-26")
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
	assert.NotEmpty(t, req.Header.Get(constants.BBHeaderOriginQC))
}
func hasContentTypeURLEncodedHeader(t *testing.T, req *http.Request) {
	assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
}
func hasURLEncodedBodyWithIdentifier(t *testing.T, req *http.Request) {
	body := reqBodyToString(req)
	assert.Contains(t, body, fmt.Sprintf("identifier=%s", url.QueryEscape("http://hl7.org/fhir/sid/us-mbi|")))
}
func hasClaimRequiredURLEncodedBody(t *testing.T, req *http.Request) {
	body := reqBodyToString(req)
	assert.Contains(t, body, "includeTaxNumbers=true")
	assert.Contains(t, body, "mbi=beneID1")
	assert.Contains(t, body, "excludeSAMHSA=true")
	assert.Contains(t, body, "isHashed=false")
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

func reqBodyToString(req *http.Request) string {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(req.Body); err != nil {
		return ""
	}
	respBytes := buf.String()
	return string(respBytes)
}

func TestBBTestSuite(t *testing.T) {
	suite.Run(t, new(BBTestSuite))
	suite.Run(t, new(BBRequestTestSuite))
}

func TestSetAppropriateServiceDateParams(t *testing.T) {
	tests := []struct {
		name         string
		params       url.Values
		expectedVals []string
	}{
		{
			name:         "No prior service date set",
			params:       url.Values{},
			expectedVals: []string(nil),
		},
		{
			name:         "Single gt val, year only",
			params:       url.Values{"service-date": []string{"gt2022"}},
			expectedVals: []string{"gt2022-01-01"},
		},
		{
			name:         "Single gt val, year and month",
			params:       url.Values{"service-date": []string{"gt2022-03"}},
			expectedVals: []string{"gt2022-03-01"},
		},
		{
			name:         "Single gt val, full date",
			params:       url.Values{"service-date": []string{"gt2022-05-05"}},
			expectedVals: []string{"gt2022-05-05"},
		},
		{
			name:         "Single le val, year only",
			params:       url.Values{"service-date": []string{"le2022"}},
			expectedVals: []string{"le2022-01-01"},
		},
		{
			name:         "Single le val, year and month",
			params:       url.Values{"service-date": []string{"le2022-03"}},
			expectedVals: []string{"le2022-03-01"},
		},
		{
			name:         "Single le val, full date",
			params:       url.Values{"service-date": []string{"le2022-05-05"}},
			expectedVals: []string{"le2022-05-05"},
		},
		{
			name:         "Single eq val, year only",
			params:       url.Values{"service-date": []string{"eq2022"}},
			expectedVals: []string{"ge2022-01-01", "lt2023-01-01"},
		},
		{
			name:         "Single eq val, year and month",
			params:       url.Values{"service-date": []string{"eq2022-03"}},
			expectedVals: []string{"ge2022-03-01", "lt2022-04-01"},
		},
		{
			name:         "Single eq val, full date",
			params:       url.Values{"service-date": []string{"eq2022-05-05"}},
			expectedVals: []string{"ge2022-05-05", "lt2022-05-06"},
		},
		{
			name:         "Single no prefix val, year only",
			params:       url.Values{"service-date": []string{"2022"}},
			expectedVals: []string{"ge2022-01-01", "lt2023-01-01"},
		},
		{
			name:         "Single no prefix val, year and month",
			params:       url.Values{"service-date": []string{"2022-03"}},
			expectedVals: []string{"ge2022-03-01", "lt2022-04-01"},
		},
		{
			name:         "Single no prefix val, full date",
			params:       url.Values{"service-date": []string{"2022-05-05"}},
			expectedVals: []string{"ge2022-05-05", "lt2022-05-06"},
		},
		{
			name: "Multiple upper and lower bounds set",
			params: url.Values{"service-date": []string{
				"gt2022-01-01",
				"ge2023-02",
				"gt2024",
				"le2022-01-01",
				"lt2023-02",
				"le2024",
				"bad time",
			}},
			expectedVals: []string{"gt2024-01-01", "le2022-01-01"},
		},
		{
			name: "Multiple upper and lower bounds set opposite ordering",
			params: url.Values{"service-date": []string{
				"gt2022",
				"ge2023-02",
				"gt2024-03-03",
				"le2022",
				"lt2023-02",
				"le2024-03-03",
				"bad time",
			}},
			expectedVals: []string{"gt2024-03-03", "le2022-01-01"},
		},
		{
			name: "Multiple upper and lower bounds set as well as eq and no prefix",
			params: url.Values{"service-date": []string{
				"gt2022",
				"ge2023-02",
				"gt2024-03-03",
				"le2022",
				"lt2023-02",
				"le2024-03-03",
				"eq2022",
				"eq2023-02",
				"eq2024-03-03",
				"2022",
				"2023-02",
				"2024-03-03",
				"20",
				"bad time",
			}},
			expectedVals: []string{"gt2024-03-03", "le2022-01-01"},
		},
		{
			name: "Multiple upper and lower bounds set as well as eq and no prefix opposite ordering",
			params: url.Values{"service-date": []string{
				"gt2022-01-01",
				"ge2023-02",
				"gt2024",
				"le2022-01-01",
				"lt2023-02",
				"le2024",
				"eq2022-01-01",
				"eq2023-02",
				"eq2024",
				"2022-01-01",
				"2023-02",
				"2024",
				"20",
				"bad time",
			}},
			expectedVals: []string{"gt2024-01-01", "le2022-01-01"},
		},
		{
			name: "Multiple eq set",
			params: url.Values{"service-date": []string{
				"eq2022-01-01",
				"eq2023-02",
				"eq2024",
				"bad time",
			}},
			expectedVals: []string{"ge2024-01-01", "lt2022-01-02"},
		},
		{
			name: "Multiple no prefix set",
			params: url.Values{"service-date": []string{
				"2022",
				"2023-02",
				"2024-03-03",
				"bad time",
			}},
			expectedVals: []string{"ge2024-03-03", "lt2023-01-01"},
		},
		{
			name: "Multiple lower bounds set",
			params: url.Values{"service-date": []string{
				"gt2022-01-01",
				"ge2022-02",
				"gt2022",
				"bad time",
			}},
			expectedVals: []string{"ge2022-02-01"},
		},
		{
			name: "Multiple upper bounds set",
			params: url.Values{"service-date": []string{
				"le2022-01-01",
				"lt2022-02",
				"le2022",
				"bad time",
			}},
			expectedVals: []string{"le2022-01-01"},
		},
		{
			name: "Multiple eq set similar dates",
			params: url.Values{"service-date": []string{
				"eq2022-01-01",
				"eq2022-01",
				"eq2022",
				"bad time",
			}},
			expectedVals: []string{"ge2022-01-01", "lt2022-01-02"},
		},
		{
			name: "Multiple no prefix set similar dates",
			params: url.Values{"service-date": []string{
				"eq2022",
				"eq2022-01",
				"eq2022-01-01",
				"bad time",
			}},
			expectedVals: []string{"ge2022-01-01", "lt2022-01-02"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedParams := &tt.params
			setAppropriateServiceDateParams(updatedParams)
			assert.Equal(t, tt.expectedVals, (*updatedParams)["service-date"])
		})
	}
}
