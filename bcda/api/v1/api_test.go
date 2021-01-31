package v1

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	fhirmodels "github.com/eug48/fhir/models"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
)

const (
	expiryHeaderFormat = "2006-01-02 15:04:05.999999999 -0700 MST"
)

var (
	acoUnderTest = uuid.Parse(constants.SmallACOUUID)
)

type APITestSuite struct {
	suite.Suite
	rr    *httptest.ResponseRecorder
	db    *sql.DB
	reset func()
}

type RequestParams struct {
	resourceType string
	since        string
	outputFormat string
}

var origDate string
var origBBCert string
var origBBKey string

func (s *APITestSuite) SetupSuite() {
	s.reset = testUtils.SetUnitTestKeysForAuth() // needed until token endpoint moves to auth
	origDate = conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", time.Now().Format("060102 15:01:01"))
	conf.SetEnv(s.T(), "BB_REQUEST_RETRY_INTERVAL_MS", "10")
	origBBCert = conf.GetEnv("BB_CLIENT_CERT_FILE")
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../../shared_files/decrypted/bfd-dev-test-cert.pem")
	origBBKey = conf.GetEnv("BB_CLIENT_KEY_FILE")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../../shared_files/decrypted/bfd-dev-test-key.pem")
}

func (s *APITestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", origDate)
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origBBCert)
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origBBKey)
	s.reset()
}

func (s *APITestSuite) SetupTest() {
	s.db = database.GetDbConnection()
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TearDownTest() {
	postgrestest.DeleteJobsByACOID(s.T(), s.db, acoUnderTest)
	s.db.Close()
}

func (s *APITestSuite) TestBulkEOBRequest() {
	since := "2020-02-13T08:00:00.000-05:00"
	bulkEOBRequestHelper("Patient", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Group/all", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkEOBRequestNoBeneficiariesInACO() {
	bulkEOBRequestNoBeneficiariesInACOHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestNoBeneficiariesInACOHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkEOBRequestMissingToken() {
	bulkEOBRequestMissingTokenHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestMissingTokenHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkPatientRequest() {
	for _, since := range []string{"", "2020-02-13T08:00:00.000-05:00", "2020-02-13T08:00:00.000+05:00"} {
		bulkPatientRequestHelper("Patient", since, s)
		s.TearDownTest()
		s.SetupTest()
		bulkPatientRequestHelper("Group/all", since, s)
		s.TearDownTest()
		s.SetupTest()
	}
}

func (s *APITestSuite) TestBulkCoverageRequest() {
	requestParams := RequestParams{}
	bulkCoverageRequestHelper("Patient", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	requestParams.since = "2020-02-13T08:00:00.000-05:00"
	bulkCoverageRequestHelper("Patient", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", requestParams, s)
}

func (s *APITestSuite) TestBulkCoverageRequestValidOutputFormatNDJSON() {
	requestParams := RequestParams{outputFormat: "ndjson"}
	bulkCoverageRequestHelper("Patient", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", requestParams, s)
}

func (s *APITestSuite) TestBulkCoverageRequestValidOutputFormatApplicationNDJSON() {
	requestParams := RequestParams{outputFormat: "application/ndjson"}
	bulkCoverageRequestHelper("Patient", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", requestParams, s)
}

func (s *APITestSuite) TestBulkCoverageRequestValidOutputFormatApplicationFHIR() {
	requestParams := RequestParams{outputFormat: "application/fhir+ndjson"}
	bulkCoverageRequestHelper("Patient", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", requestParams, s)
}

func (s *APITestSuite) TestBulkCoverageRequestValidMultipleParameters() {
	requestParams := RequestParams{outputFormat: "application/fhir+ndjson", since: "2020-02-13T08:00:00.000-05:00"}
	bulkCoverageRequestHelper("Patient", requestParams, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", requestParams, s)
}

func (s *APITestSuite) TestBulkConcurrentRequest() {
	bulkConcurrentRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkConcurrentRequestHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkConcurrentRequestTime() {
	bulkConcurrentRequestTimeHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkConcurrentRequestTimeHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkPatientRequestBBClientFailure() {
	bulkPatientRequestBBClientFailureHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestBBClientFailureHelper("Group/all", s)
}

func bulkEOBRequestHelper(endpoint, since string, s *APITestSuite) {
	acoID := acoUnderTest
	requestParams := RequestParams{resourceType: "ExplanationOfBenefit", since: since}
	_, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)
	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
}

func bulkEOBRequestNoBeneficiariesInACOHelper(endpoint string, s *APITestSuite) {
	acoID := uuid.Parse("A40404F7-1EF2-485A-9B71-40FE7ACDCBC2")
	jobCount := s.getJobCount(acoID)

	// Since we should've failed somewhere in the bulk request, we should not have
	// have a job count change.
	defer s.verifyJobCount(acoID, jobCount)

	requestParams := RequestParams{resourceType: "ExplanationOfBenefit"}
	_, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)
	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)
}

func bulkEOBRequestMissingTokenHelper(endpoint string, s *APITestSuite) {
	requestParams := RequestParams{resourceType: "ExplanationOfBenefit"}
	_, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.TokenErr, respOO.Issue[0].Details.Coding[0].Code)
}

func bulkPatientRequestHelper(endpoint, since string, s *APITestSuite) {
	acoID := acoUnderTest

	requestParams := RequestParams{resourceType: "Patient", since: since}
	_, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)

	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
}

func bulkCoverageRequestHelper(endpoint string, requestParams RequestParams, s *APITestSuite) {
	acoID := acoUnderTest

	requestParams.resourceType = "Coverage"
	_, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)

	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
}

func bulkPatientRequestBBClientFailureHelper(endpoint string, s *APITestSuite) {
	orig := conf.GetEnv("BB_CLIENT_CERT_FILE")
	defer conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", orig)

	err := conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "blah")
	assert.Nil(s.T(), err)

	acoID := acoUnderTest

	jobCount := s.getJobCount(acoID)

	// Since we should've failed somewhere in the bulk request, we should not have
	// have a job count change.
	defer s.verifyJobCount(acoID, jobCount)

	requestParams := RequestParams{resourceType: "Patient"}
	_, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)
	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handlerFunc(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)
}

func bulkConcurrentRequestHelper(endpoint string, s *APITestSuite) {
	err := conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", "prod")
	defer conf.UnsetEnv(s.T(), "DEPLOYMENT_TARGET")
	assert.Nil(s.T(), err)
	acoID := acoUnderTest

	firstRequestParams := RequestParams{resourceType: "ExplanationOfBenefit"}
	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, firstRequestParams)

	j := models.Job{
		ACOID:      acoID,
		RequestURL: requestUrl,
		Status:     models.JobStatusInProgress,
		JobCount:   1,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	tests := []struct {
		status        models.JobStatus
		expStatusCode int
	}{
		{models.JobStatusPending, http.StatusTooManyRequests},
		{models.JobStatusInProgress, http.StatusTooManyRequests},
		{models.JobStatusCompleted, http.StatusAccepted},
		{models.JobStatusArchived, http.StatusAccepted},
		{models.JobStatusExpired, http.StatusAccepted},
		{models.JobStatusFailed, http.StatusAccepted},
		{models.JobStatusCancelled, http.StatusAccepted},
		{models.JobStatusFailedExpired, http.StatusAccepted},
	}
	assert.Equal(s.T(), len(models.AllJobStatuses), len(tests), "Not all models.JobStatus tested.")

	exp := regexp.MustCompile(`\/api\/v1\/jobs\/(\d+)`)
	for _, tt := range tests {
		s.T().Run(string(tt.status), func(t *testing.T) {
			j.Status = tt.status
			postgrestest.UpdateJob(s.T(), s.db, j)

			rr := httptest.NewRecorder()
			handlerFunc(rr, req)

			assert.Equal(t, tt.expStatusCode, rr.Code)
			// If we've created a job, we need to make sure it's removed to not affect other tests
			if rr.Code == http.StatusAccepted {
				jobID, err := strconv.ParseUint(exp.FindStringSubmatch(rr.Header().Get("Content-Location"))[1], 10, 64)
				assert.NoError(t, err)
				postgrestest.DeleteJobByID(t, s.db, uint(jobID))
			}
		})
	}

	// different aco same endpoint
	j.ACOID, j.Status = uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"), models.JobStatusInProgress
	postgrestest.UpdateJob(s.T(), s.db, j)
	rr := httptest.NewRecorder()
	handlerFunc(rr, req)
	assert.Equal(s.T(), http.StatusAccepted, rr.Code)
	jobID, err := strconv.ParseUint(exp.FindStringSubmatch(rr.Header().Get("Content-Location"))[1], 10, 64)
	assert.NoError(s.T(), err)
	postgrestest.DeleteJobByID(s.T(), s.db, uint(jobID))
}

func bulkConcurrentRequestTimeHelper(endpoint string, s *APITestSuite) {
	err := conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", "prod")
	defer conf.UnsetEnv(s.T(), "DEPLOYMENT_TARGET")
	assert.Nil(s.T(), err)
	acoID := acoUnderTest

	requestParams := RequestParams{resourceType: "ExplanationOfBenefit"}
	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, requestParams)

	j := models.Job{
		ACOID:      acoID,
		RequestURL: requestUrl,
		Status:     models.JobStatusInProgress,
		JobCount:   1,
	}

	postgrestest.CreateJobs(s.T(), s.db, &j)

	ad := s.makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	// serve job
	handler := http.HandlerFunc(handlerFunc)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusTooManyRequests, rr.Code)

	// change created_at timestamp so that the job is considered expired.
	// Use an offset to account for any clock skew
	j.CreatedAt = j.CreatedAt.Add(-(api.GetJobTimeout() + time.Second))
	postgrestest.UpdateJob(s.T(), s.db, j)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusAccepted, rr.Code)
}

func bulkRequestHelper(endpoint string, testRequestParams RequestParams) (string, func(http.ResponseWriter, *http.Request), *http.Request) {
	var handlerFunc http.HandlerFunc
	var req *http.Request
	var group string

	if endpoint == "Patient" {
		handlerFunc = BulkPatientRequest
	} else if endpoint == "Group/all" {
		handlerFunc = BulkGroupRequest
		group = "all"
	}

	requestUrl, _ := url.Parse(fmt.Sprintf("/api/v1/%s/$export", endpoint))
	q := requestUrl.Query()
	if testRequestParams.resourceType != "" {
		q.Set("_type", testRequestParams.resourceType)
	}
	if testRequestParams.since != "" {
		q.Set("_since", testRequestParams.since)
	}
	if testRequestParams.outputFormat != "" {
		q.Set("_outputFormat", testRequestParams.outputFormat)
	}

	requestUrl.RawQuery = q.Encode()
	req = httptest.NewRequest("GET", requestUrl.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", group)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return requestUrl.Path, handlerFunc, req
}

func (s *APITestSuite) TestJobStatusBadInputs() {
	tests := []struct {
		name          string
		jobID         string
		expStatusCode int
		expErrCode    string
	}{
		{"InvalidJobID", "abcd", 400, responseutils.RequestErr},
		{"DoesNotExist", "0", 404, responseutils.DbErr},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", tt.jobID), nil)
			rr := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			ad := s.makeContextValues(acoUnderTest)
			req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

			JobStatus(rr, req)

			var respOO fhirmodels.OperationOutcome
			err := json.Unmarshal(rr.Body.Bytes(), &respOO)
			assert.NoError(t, err)

			assert.Equal(t, responseutils.Error, respOO.Issue[0].Severity)
			assert.Equal(t, responseutils.Exception, respOO.Issue[0].Code)
			assert.Equal(t, tt.expErrCode, respOO.Issue[0].Details.Coding[0].Code)
		})
	}
}

func (s *APITestSuite) TestJobStatusNotComplete() {
	tests := []struct {
		status        models.JobStatus
		expStatusCode int
	}{
		{models.JobStatusPending, http.StatusAccepted},
		{models.JobStatusInProgress, http.StatusAccepted},
		{models.JobStatusFailed, http.StatusInternalServerError},
		{models.JobStatusFailedExpired, http.StatusInternalServerError},
		{models.JobStatusExpired, http.StatusGone},
		{models.JobStatusArchived, http.StatusGone},
		{models.JobStatusCancelled, http.StatusNotFound},
	}

	for _, tt := range tests {
		s.T().Run(string(tt.status), func(t *testing.T) {
			j := models.Job{
				ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
				RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
				Status:     tt.status,
			}
			postgrestest.CreateJobs(t, s.db, &j)
			defer postgrestest.DeleteJobByID(t, s.db, j.ID)

			req := s.createJobStatusRequest(acoUnderTest, j.ID)
			rr := httptest.NewRecorder()

			JobStatus(rr, req)
			assert.Equal(t, tt.expStatusCode, rr.Code)
			if rr.Code == http.StatusAccepted {
				assert.Contains(t, rr.Header().Get("X-Progress"), tt.status)
				assert.Equal(t, "", rr.Header().Get("Expires"))
			} else if rr.Code == http.StatusInternalServerError {
				assert.Contains(t, rr.Body.String(), "Service encountered numerous errors")
			} else if rr.Code == http.StatusGone {
				assertExpiryEquals(t, j.CreatedAt.Add(api.GetJobTimeout()), rr.Header().Get("Expires"))
			}
		})
	}
}

// https://stackoverflow.com/questions/34585957/postgresql-9-3-how-to-insert-upper-case-uuid-into-table
func (s *APITestSuite) TestJobStatusCompleted() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	var expectedUrls []string
	for i := 1; i <= 10; i++ {
		fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
		expectedurl := fmt.Sprintf("%s/%s/%s", "http://example.com/data", fmt.Sprint(j.ID), fileName)
		expectedUrls = append(expectedUrls, expectedurl)
		postgrestest.CreateJobKeys(s.T(), s.db,
			models.JobKey{JobID: j.ID, FileName: fileName, ResourceType: "ExplanationOfBenefit"})
	}

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), "application/json", s.rr.Header().Get("Content-Type"))
	str := s.rr.Header().Get("Expires")
	fmt.Println(str)
	assertExpiryEquals(s.T(), j.CreatedAt.Add(api.GetJobTimeout()), s.rr.Header().Get("Expires"))

	var rb api.BulkResponseBody
	err := json.Unmarshal(s.rr.Body.Bytes(), &rb)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), j.RequestURL, rb.RequestURL)
	assert.Equal(s.T(), true, rb.RequiresAccessToken)
	assert.Equal(s.T(), "ExplanationOfBenefit", rb.Files[0].Type)
	assert.Equal(s.T(), len(expectedUrls), len(rb.Files))
	// Order of these values is impossible to know so this is the only way
	for _, fileItem := range rb.Files {
		inOutput := false
		for _, expectedUrl := range expectedUrls {
			if fileItem.URL == expectedUrl {
				inOutput = true
				break
			}
		}
		assert.True(s.T(), inOutput)

	}
	assert.Empty(s.T(), rb.Errors)
}

func (s *APITestSuite) TestJobStatusCompletedErrorFileExists() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
	jobKey := models.JobKey{
		JobID:        j.ID,
		FileName:     fileName,
		ResourceType: "ExplanationOfBenefit",
	}
	postgrestest.CreateJobKeys(s.T(), s.db, jobKey)

	f := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_PAYLOAD_DIR"), fmt.Sprint(j.ID))
	if _, err := os.Stat(f); os.IsNotExist(err) {
		err = os.MkdirAll(f, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	errFileName := strings.Split(jobKey.FileName, ".")[0]
	errFilePath := fmt.Sprintf("%s/%s/%s-error.ndjson", conf.GetEnv("FHIR_PAYLOAD_DIR"), fmt.Sprint(j.ID), errFileName)
	_, err := os.Create(errFilePath)
	if err != nil {
		s.T().Error(err)
	}

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), "application/json", s.rr.Header().Get("Content-Type"))

	var rb api.BulkResponseBody
	err = json.Unmarshal(s.rr.Body.Bytes(), &rb)
	if err != nil {
		s.T().Error(err)
	}

	dataurl := fmt.Sprintf("%s/%s/%s", "http://example.com/data", fmt.Sprint(j.ID), fileName)
	errorurl := fmt.Sprintf("%s/%s/%s-error.ndjson", "http://example.com/data", fmt.Sprint(j.ID), errFileName)

	assert.Equal(s.T(), j.RequestURL, rb.RequestURL)
	assert.Equal(s.T(), true, rb.RequiresAccessToken)
	assert.Equal(s.T(), "ExplanationOfBenefit", rb.Files[0].Type)
	assert.Equal(s.T(), dataurl, rb.Files[0].URL)
	assert.Equal(s.T(), "OperationOutcome", rb.Errors[0].Type)
	assert.Equal(s.T(), errorurl, rb.Errors[0].URL)

	os.Remove(errFilePath)
}

// This job is old, but has not yet been marked as expired.
func (s *APITestSuite) TestJobStatusNotExpired() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	j.UpdatedAt = time.Now().Add(-(api.GetJobTimeout() + time.Second))
	postgrestest.UpdateJob(s.T(), s.db, j)

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	assertExpiryEquals(s.T(), j.UpdatedAt.Add(api.GetJobTimeout()), s.rr.Header().Get("Expires"))
}

func (s *APITestSuite) TestServeData() {
	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", "../../../bcdaworker/data/test")

	tests := []struct {
		name    string
		headers []string
	}{
		{"gzip-only", []string{"gzip"}},
		{"gzip", []string{"deflate", "br", "gzip"}},
		{"non-gzip", nil},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			defer s.SetupTest()
			req := httptest.NewRequest("GET", "/data/test.ndjson", nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("fileName", "test.ndjson")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			var useGZIP bool
			for _, h := range tt.headers {
				req.Header.Add("Accept-Encoding", h)
				if h == "gzip" {
					useGZIP = true
				}
			}

			handler := http.HandlerFunc(ServeData)
			handler.ServeHTTP(s.rr, req)

			assert.Equal(t, http.StatusOK, s.rr.Code)
			assert.Equal(t, "application/fhir+ndjson", s.rr.Result().Header.Get("Content-Type"))

			var b []byte

			if useGZIP {
				assert.Equal(t, "gzip", s.rr.Header().Get("Content-Encoding"))
				reader, err := gzip.NewReader(s.rr.Body)
				assert.NoError(t, err)
				defer reader.Close()
				b, err = ioutil.ReadAll(reader)
				assert.NoError(t, err)
			} else {
				assert.Equal(t, "", s.rr.Header().Get("Content-Encoding"))
				b = s.rr.Body.Bytes()
			}

			assert.Contains(t, string(b), `{"resourceType": "Bundle", "total": 33, "entry": [{"resource": {"status": "active", "diagnosis": [{"diagnosisCodeableConcept": {"coding": [{"system": "http://hl7.org/fhir/sid/icd-9-cm", "code": "2113"}]},`)

		})
	}
}

func (s *APITestSuite) TestMetadata() {
	req := httptest.NewRequest("GET", "/api/v1/metadata", nil)
	req.TLS = &tls.ConnectionState{}

	handler := http.HandlerFunc(Metadata)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}

func (s *APITestSuite) TestGetVersion() {
	req := httptest.NewRequest("GET", "/_version", nil)

	handler := http.HandlerFunc(GetVersion)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)

	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}

	assert.Equal(s.T(), "latest", respMap["version"])
}

func (s *APITestSuite) TestJobStatusWithWrongACO() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     models.JobStatusPending,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	handler := auth.RequireTokenJobMatch(http.HandlerFunc(JobStatus))
	req := s.createJobStatusRequest(uuid.Parse(constants.LargeACOUUID), j.ID)

	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
}

func (s *APITestSuite) TestHealthCheck() {
	req, err := http.NewRequest("GET", "/_health", nil)
	assert.Nil(s.T(), err)
	handler := http.HandlerFunc(HealthCheck)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}

func (s *APITestSuite) TestHealthCheckWithBadDatabaseURL() {
	// Mock database.LogFatal() to allow execution to continue despite bad URL
	origLogFatal := database.LogFatal
	defer func() { database.LogFatal = origLogFatal }()
	database.LogFatal = func(args ...interface{}) {
		fmt.Println("FATAL (NO-OP)")
	}
	dbURL := conf.GetEnv("DATABASE_URL")
	defer conf.SetEnv(s.T(), "DATABASE_URL", dbURL)
	conf.SetEnv(s.T(), "DATABASE_URL", "not-a-database")
	req, err := http.NewRequest("GET", "/_health", nil)
	assert.Nil(s.T(), err)
	handler := http.HandlerFunc(HealthCheck)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadGateway, s.rr.Code)
}

func (s *APITestSuite) TestAuthInfoDefault() {

	// get original provider so we can reset at the end of the test
	originalProvider := auth.GetProviderName()

	// set provider to bogus value and make sure default (alpha) is retrieved
	auth.SetProvider("bogus")
	req := httptest.NewRequest("GET", "/_auth", nil)
	handler := http.HandlerFunc(GetAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}
	assert.Equal(s.T(), "alpha", respMap["auth_provider"])

	// set provider back to original value
	auth.SetProvider(originalProvider)
}

func (s *APITestSuite) TestAuthInfoAlpha() {

	// get original provider so we can reset at the end of the test
	originalProvider := auth.GetProviderName()

	// set provider to alpha and make sure alpha is retrieved
	auth.SetProvider("alpha")
	req := httptest.NewRequest("GET", "/_auth", nil)
	handler := http.HandlerFunc(GetAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}
	assert.Equal(s.T(), "alpha", respMap["auth_provider"])

	// set provider back to original value
	auth.SetProvider(originalProvider)
}

func (s *APITestSuite) TestAuthInfoOkta() {

	// get original provider so we can reset at the end of the test
	originalProvider := auth.GetProviderName()

	// set provider to okta and make sure okta is retrieved
	auth.SetProvider("okta")
	req := httptest.NewRequest("GET", "/_auth", nil)
	handler := http.HandlerFunc(GetAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}
	assert.Equal(s.T(), "okta", respMap["auth_provider"])

	// set provider back to original value
	auth.SetProvider(originalProvider)
}

func (s *APITestSuite) verifyJobCount(acoID uuid.UUID, expectedJobCount int) {
	count := s.getJobCount(acoID)
	assert.Equal(s.T(), expectedJobCount, count)
}

func (s *APITestSuite) getJobCount(acoID uuid.UUID) int {
	jobs := postgrestest.GetJobsByACOID(s.T(), s.db, acoID)
	return len(jobs)
}

func (s *APITestSuite) makeContextValues(acoID uuid.UUID) (data auth.AuthData) {
	aco := postgrestest.GetACOByUUID(s.T(), s.db, acoID)
	return auth.AuthData{ACOID: aco.UUID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
}

func (s *APITestSuite) createJobStatusRequest(acoID uuid.UUID, jobID uint) *http.Request {
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", jobID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(jobID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := s.makeContextValues(acoID)
	return req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

// Compare expiry header against the expected time value.
// There seems to be some slight difference in precision here,
// so we'll compare up to seconds
func assertExpiryEquals(t *testing.T, expectedTime time.Time, expiry string) {
	expiryTime, err := time.Parse(expiryHeaderFormat, expiry)
	if err != nil {
		t.Fatalf("Failed to parse %s to time.Time %s", expiry, err)
	}

	assert.Equal(t, time.Duration(0), expectedTime.Round(time.Second).Sub(expiryTime.Round(time.Second)))
}
