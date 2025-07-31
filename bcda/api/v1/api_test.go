package v1

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

const (
	expiryHeaderFormat = "2006-01-02 15:04:05.999999999 -0700 MST"
)

var (
	acoUnderTest = uuid.Parse(constants.SmallACOUUID)
)

type APITestSuite struct {
	suite.Suite
	rr         *httptest.ResponseRecorder
	connection *sql.DB
	pool       *pgxv5Pool.Pool
	provider   auth.Provider
	apiV1      *ApiV1
}

func (s *APITestSuite) SetupSuite() {
	s.connection = database.GetConnection()
	s.provider = auth.NewProvider(s.connection)
	s.apiV1 = NewApiV1(s.connection, s.pool, s.provider)

	origDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", time.Now().Format("060102 15:01:01"))
	conf.SetEnv(s.T(), "BB_REQUEST_RETRY_INTERVAL_MS", "10")
	origBBCert := conf.GetEnv("BB_CLIENT_CERT_FILE")
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../../shared_files/decrypted/bfd-dev-test-cert.pem")
	origBBKey := conf.GetEnv("BB_CLIENT_KEY_FILE")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../../shared_files/decrypted/bfd-dev-test-key.pem")

	s.T().Cleanup(func() {
		conf.SetEnv(s.T(), "CCLF_REF_DATE", origDate)
		conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origBBCert)
		conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origBBKey)
	})
}

func (s *APITestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TearDownTest() {
	postgrestest.DeleteJobsByACOID(s.T(), s.connection, acoUnderTest)
}

func (s *APITestSuite) TestJobStatusBadInputs() {
	tests := []struct {
		name          string
		jobID         string
		expStatusCode int
		expErrCode    string
	}{
		{"InvalidJobID", "abcd", 400, "could not parse job id"},
		{"DoesNotExist", "0", 404, "Job not found."},
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
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))

			s.apiV1.JobStatus(rr, req)

			respOO := getOperationOutcome(t, rr.Body.Bytes())

			assert.Equal(t, fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
			assert.Equal(t, fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
			assert.Equal(t, tt.expErrCode, respOO.Issue[0].Diagnostics.Value)
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
		{models.JobStatusCancelledExpired, http.StatusNotFound},
	}

	for _, tt := range tests {
		s.T().Run(string(tt.status), func(t *testing.T) {
			j := models.Job{
				ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
				RequestURL: constants.V1Path + constants.PatientEOBPath,
				Status:     tt.status,
			}
			postgrestest.CreateJobs(t, s.connection, &j)
			defer postgrestest.DeleteJobByID(t, s.connection, j.ID)

			req := s.createJobStatusRequest(acoUnderTest, j.ID)
			rr := httptest.NewRecorder()

			s.apiV1.JobStatus(rr, req)
			assert.Equal(t, tt.expStatusCode, rr.Code)
			switch rr.Code {
			case http.StatusAccepted:
				assert.Contains(t, rr.Header().Get("X-Progress"), tt.status)
				assert.Equal(t, "", rr.Header().Get("Expires"))
			case http.StatusInternalServerError:
				assert.Contains(t, rr.Body.String(), "Service encountered numerous errors")
			case http.StatusGone:
				assertExpiryEquals(t, j.CreatedAt.Add(s.apiV1.handler.JobTimeout), rr.Header().Get("Expires"))
			}
		})
	}
}

// https://stackoverflow.com/questions/34585957/postgresql-9-3-how-to-insert-upper-case-uuid-into-table
func (s *APITestSuite) TestJobStatusCompleted() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)

	var expectedUrls []string
	for i := 1; i <= 10; i++ {
		fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
		expectedurl := fmt.Sprintf("%s/%s/%s", constants.ExpectedTestUrl, fmt.Sprint(j.ID), fileName)
		expectedUrls = append(expectedUrls, expectedurl)
		postgrestest.CreateJobKeys(s.T(), s.connection,
			models.JobKey{JobID: j.ID, FileName: fileName, ResourceType: "ExplanationOfBenefit"})
	}

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	s.apiV1.JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), "application/json", s.rr.Header().Get(constants.ContentType))
	str := s.rr.Header().Get("Expires")
	fmt.Println(str)
	assertExpiryEquals(s.T(), j.CreatedAt.Add(s.apiV1.handler.JobTimeout), s.rr.Header().Get("Expires"))

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
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)

	fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
	jobKey := models.JobKey{
		JobID:        j.ID,
		FileName:     fileName,
		ResourceType: "ExplanationOfBenefit",
	}
	postgrestest.CreateJobKeys(s.T(), s.connection, jobKey)

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
	s.apiV1.JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), "application/json", s.rr.Header().Get(constants.ContentType))

	var rb api.BulkResponseBody
	err = json.Unmarshal(s.rr.Body.Bytes(), &rb)
	if err != nil {
		s.T().Error(err)
	}

	dataurl := fmt.Sprintf("%s/%s/%s", constants.ExpectedTestUrl, fmt.Sprint(j.ID), fileName)
	errorurl := fmt.Sprintf("%s/%s/%s-error.ndjson", constants.ExpectedTestUrl, fmt.Sprint(j.ID), errFileName)

	assert.Equal(s.T(), j.RequestURL, rb.RequestURL)
	assert.Equal(s.T(), true, rb.RequiresAccessToken)
	assert.Equal(s.T(), "ExplanationOfBenefit", rb.Files[0].Type)
	assert.Equal(s.T(), dataurl, rb.Files[0].URL)
	for _, file := range rb.Files {
		assert.NotContains(s.T(), file.URL, "-error.ndjson")
	}
	assert.Equal(s.T(), "OperationOutcome", rb.Errors[0].Type)
	assert.Equal(s.T(), errorurl, rb.Errors[0].URL)

	os.Remove(errFilePath)
}

// This job is old, but has not yet been marked as expired.
func (s *APITestSuite) TestJobStatusNotExpired() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)

	j.UpdatedAt = time.Now().Add(-(s.apiV1.handler.JobTimeout + time.Second))
	postgrestest.UpdateJob(s.T(), s.connection, j)

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	s.apiV1.JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	assertExpiryEquals(s.T(), j.UpdatedAt.Add(s.apiV1.handler.JobTimeout), s.rr.Header().Get("Expires"))
}

func (s *APITestSuite) TestDeleteJobBadInputs() {
	tests := []struct {
		name          string
		jobID         string
		expStatusCode int
		expErrCode    string
	}{
		{"InvalidJobID", "abcd", 400, "could not parse job id"},
		{"DoesNotExist", "0", 404, "Job not found."},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/jobs/%s", tt.jobID), nil)
			rr := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			ad := s.makeContextValues(acoUnderTest)
			req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))

			s.apiV1.JobStatus(rr, req)

			respOO := getOperationOutcome(t, rr.Body.Bytes())

			assert.Equal(t, fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
			assert.Equal(t, fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
			assert.Contains(t, respOO.Issue[0].Diagnostics.Value, tt.expErrCode)
		})
	}
}

func (s *APITestSuite) TestDeleteJob() {
	tests := []struct {
		status        models.JobStatus
		expStatusCode int
	}{
		{models.JobStatusPending, http.StatusAccepted},
		{models.JobStatusInProgress, http.StatusAccepted},
		{models.JobStatusFailed, http.StatusGone},
		{models.JobStatusExpired, http.StatusGone},
		{models.JobStatusArchived, http.StatusGone},
		{models.JobStatusCompleted, http.StatusGone},
		{models.JobStatusCancelled, http.StatusGone},
		{models.JobStatusFailedExpired, http.StatusGone},
	}

	for _, tt := range tests {
		s.T().Run(string(tt.status), func(t *testing.T) {
			j := models.Job{
				ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
				RequestURL: constants.V1Path + constants.PatientEOBPath,
				Status:     tt.status,
			}
			postgrestest.CreateJobs(t, s.connection, &j)
			defer postgrestest.DeleteJobByID(t, s.connection, j.ID)

			req := s.createJobStatusRequest(acoUnderTest, j.ID)
			rr := httptest.NewRecorder()

			s.apiV1.DeleteJob(rr, req)
			assert.Equal(t, tt.expStatusCode, rr.Code)
			if rr.Code == http.StatusGone {
				assert.Contains(t, rr.Body.String(), "job was not cancelled because it is not Pending or In Progress")
			}
		})
	}
}

func (s *APITestSuite) TestServeData() {
	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", "../../../shared_files/gzip_feature_test/")

	tests := []struct {
		name         string
		headers      []string
		gzipExpected bool
		fileName     string
		validFile    bool
	}{
		{"no-header-gzip-encoded", []string{""}, false, "test_gzip_encoded.ndjson", true},
		{"yes-header-gzip-encoded", []string{"gzip"}, true, "test_gzip_encoded.ndjson", true},
		{"yes-header-not-encoded", []string{"gzip"}, true, "test_no_encoding.ndjson", true},
		{"bad file name", []string{""}, false, "not_a_real_file", false},
		{"single byte file", []string{""}, false, "single_byte_file.bin", false},
		{"no-header-corrupt-file", []string{""}, false, "corrupt_gz_file.ndjson", false}, //This file is kind of cool. has magic number, but otherwise arbitrary data.
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			defer s.SetupTest()
			req := httptest.NewRequest("GET", fmt.Sprintf("/data/%s", tt.fileName), nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("fileName", tt.fileName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			var useGZIP bool
			for _, h := range tt.headers {
				req.Header.Add("Accept-Encoding", h)
				if strings.Contains(h, "gzip") {
					useGZIP = true
				}
			}
			assert.Equal(t, tt.gzipExpected, useGZIP)

			handler := http.HandlerFunc(ServeData)
			handler.ServeHTTP(s.rr, req)

			if !tt.validFile {
				assert.Equal(t, http.StatusInternalServerError, s.rr.Code)
				return
			}

			assert.Equal(t, http.StatusOK, s.rr.Code)
			assert.Equal(t, "application/fhir+ndjson", s.rr.Result().Header.Get(constants.ContentType))

			var b []byte

			if useGZIP {
				assert.Equal(t, "gzip", s.rr.Header().Get("Content-Encoding"))
				reader, err := gzip.NewReader(s.rr.Body)
				assert.NoError(t, err)
				defer reader.Close()
				b, err = io.ReadAll(reader)
				assert.NoError(t, err)
			} else {
				assert.Equal(t, "", s.rr.Header().Get("Content-Encoding"))
				b = s.rr.Body.Bytes()
			}

			assert.Contains(t, string(b), `{"billablePeriod":{"end":"2016-10-30","start":"2016-10-30"},"contained":[{"birthDate":"1953-05-18","gender":"male","id":"patient","identifier":[{"system":"http://hl7.org/fhir/sid/us-mbi","type":{"coding":[{"code":"MC","display":"Patient's Medicare Number","system":"http://terminology.hl7.org/CodeSystem/v2-0203"}]},"value":"1S00E00KW61"}],"name":[{"family":"Quitzon246","given":["Rodolfo763"],"text":"Rodolfo763 Quitzon246 ([max 10 chars of first], [max 15 chars of last])"}],"resourceType":"Patient"},{"id":"provider-org","identifier":[{"system":"https://bluebutton.cms.gov/resources/variables/fiss/meda-prov-6","type":{"coding":[{"code":"PRN","display":"Provider number","system":"http://terminology.hl7.org/CodeSystem/v2-0203"}]},"value":"450702"},{"system":"https://bluebutton.cms.gov/resources/variables/fiss/fed-tax-nb","type":{"coding":[{"code":"TAX","display":"Tax ID number","system":"http://terminology.hl7.org/CodeSystem/v2-0203"}]},"value":"XX-XXXXXXX"},{"system":"http://hl7.org/fhir/sid/us-npi","type":{"coding":[{"code":"npi","display":"National Provider Identifier","system":"http://hl7.org/fhir/us/carin-bb/CodeSystem/C4BBIdentifierType"}]},"value":"8884381863"}],"resourceType":"Organization"}],"created":"2024-04-19T11:37:21-04:00","diagnosis":[{"diagnosisCodeableConcept":{"coding":[{"code":"P292","system":"http://hl7.org/fhir/sid/icd-10-cm"}]},"sequence":0}],"extension":[{"url":"https://bluebutton.cms.gov/resources/variables/fiss/serv-typ-cd","valueCoding":{"code":"2","system":"https://bluebutton.cms.gov/resources/variables/fiss/serv-typ-cd"}}],"facility":{"extension":[{"url":"https://bluebutton.cms.gov/resources/variables/fiss/lob-cd","valueCoding":{"code":"2","system":"https://bluebutton.cms.gov/resources/variables/fiss/lob-cd"}}]},"id":"f-LTEwMDE5NTYxNA","identifier":[{"system":"https://bluebutton.cms.gov/resources/variables/fiss/dcn","type":{"coding":[{"code":"uc","display":"Unique Claim ID","system":"http://hl7.org/fhir/us/carin-bb/CodeSystem/C4BBIdentifierType"}]},"value":"dcn-100195614"}],"insurance":[{"coverage":{"identifier":{"system":"https://bluebutton.cms.gov/resources/variables/fiss/payers-name","value":"MEDICARE"}},"focal":true,"sequence":0}],"meta":{"lastUpdated":"2023-04-13T18:18:27.334-04:00"},"patient":{"reference":"#patient"},"priority":{"coding":[{"code":"normal","display":"Normal","system":"http://terminology.hl7.org/CodeSystem/processpriority"}]},"provider":{"reference":"#provider-org"},"resourceType":"Claim","status":"active","supportingInfo":[{"category":{"coding":[{"code":"typeofbill","display":"Type of Bill","system":"http://hl7.org/fhir/us/carin-bb/CodeSystem/C4BBSupportingInfoType"}]},"code":{"coding":[{"code":"1","system":"https://bluebutton.cms.gov/resources/variables/fiss/freq-cd"}]},"sequence":1}],"total":{"currency":"USD","value":639.66},"type":{"coding":[{"code":"institutional","display":"Institutional","system":"http://terminology.hl7.org/CodeSystem/claim-type"}]},"use":"claim"}`)

		})
	}
}

func (s *APITestSuite) TestMetadata() {
	req := httptest.NewRequest("GET", "/api/v1/metadata", nil)
	req.TLS = &tls.ConnectionState{}

	handler := http.HandlerFunc(s.apiV1.Metadata)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}

func (s *APITestSuite) TestGetVersion() {
	req := httptest.NewRequest("GET", "/_version", nil)

	handler := http.HandlerFunc(s.apiV1.GetVersion)
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
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusPending,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)

	handler := auth.RequireTokenJobMatch(s.connection)(http.HandlerFunc(s.apiV1.JobStatus))
	req := s.createJobStatusRequest(uuid.Parse(constants.LargeACOUUID), j.ID)

	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)
}

func (s *APITestSuite) TestJobsStatus() {
	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.connection, j.ID)

	s.apiV1.JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)
}

func (s *APITestSuite) TestJobsStatusNotFound() {
	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	s.apiV1.JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusNotFound, rr.Code)
}

func (s *APITestSuite) TestJobsStatusNotFoundWithStatus() {
	req := httptest.NewRequest("GET", "/api/v1/jobs?_status=Failed", nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.connection, j.ID)

	s.apiV1.JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusNotFound, rr.Code)
}

func (s *APITestSuite) TestJobsStatusWithStatus() {
	req := httptest.NewRequest("GET", "/api/v1/jobs?_status=Failed", nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusFailed,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.connection, j.ID)

	s.apiV1.JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)
}

func (s *APITestSuite) TestJobsStatusWithStatuses() {
	req := httptest.NewRequest("GET", "/api/v1/jobs?_status=Completed,Failed", nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V1Path + constants.PatientEOBPath,
		Status:     models.JobStatusFailed,
	}
	postgrestest.CreateJobs(s.T(), s.connection, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.connection, j.ID)

	s.apiV1.JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)
}

func (s *APITestSuite) TestHealthCheck() {
	req, err := http.NewRequest("GET", "/_health", nil)
	assert.Nil(s.T(), err)
	handler := http.HandlerFunc(s.apiV1.HealthCheck)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}
func (s *APITestSuite) TestAuthInfo() {
	req, err := http.NewRequest("GET", "/_auth", nil)
	assert.Nil(s.T(), err)
	handler := http.HandlerFunc(s.apiV1.GetAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)

	var resp map[string]string
	err = json.Unmarshal(s.rr.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "ssas", resp["auth_provider"])
}

func (s *APITestSuite) TestGetAttributionStatus() {
	req := httptest.NewRequest("GET", "/api/v1/attribution_status", nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	s.apiV1.AttributionStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)

	var resp api.AttributionFileStatusResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)

	aco := postgrestest.GetACOByUUID(s.T(), s.connection, acoUnderTest)
	cclfFile := postgrestest.GetLatestCCLFFileByCMSIDAndType(s.T(), s.connection, *aco.CMSID, models.FileTypeDefault)

	assert.Equal(s.T(), "last_attribution_update", resp.Data[0].Type)
	assert.Equal(s.T(), cclfFile.Timestamp.Format("2006-01-02 15:04:05"), resp.Data[0].Timestamp.Format("2006-01-02 15:04:05"))
}

func (s *APITestSuite) makeContextValues(acoID uuid.UUID) (data auth.AuthData) {
	aco := postgrestest.GetACOByUUID(s.T(), s.connection, acoID)
	return auth.AuthData{ACOID: aco.UUID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
}

func (s *APITestSuite) createJobStatusRequest(acoID uuid.UUID, jobID uint) *http.Request {
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", jobID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(jobID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
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

func getOperationOutcome(t *testing.T, data []byte) *fhirmodels.OperationOutcome {
	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.STU3)
	assert.NoError(t, err)
	container, err := unmarshaller.Unmarshal(data)
	assert.NoError(t, err)
	return container.(*fhirmodels.ContainedResource).GetOperationOutcome()
}

func MakeTestStructuredLoggerEntry(logFields logrus.Fields) *log.StructuredLoggerEntry {
	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logFields)}
	return newLogEntry
}
