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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
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
	rr *httptest.ResponseRecorder
	db *sql.DB
}

func (s *APITestSuite) SetupSuite() {
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
		testUtils.SetUnitTestKeysForAuth() // needed until token endpoint moves to auth
	})
}

func (s *APITestSuite) SetupTest() {
	s.db = database.Connection
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TearDownTest() {
	postgrestest.DeleteJobsByACOID(s.T(), s.db, acoUnderTest)
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

			respOO := getOperationOutcome(t, rr.Body.Bytes())

			assert.Equal(t, fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
			assert.Equal(t, fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
			assert.Equal(t, tt.expErrCode, respOO.Issue[0].Details.Coding[0].Code.Value)
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
				assertExpiryEquals(t, j.CreatedAt.Add(h.JobTimeout), rr.Header().Get("Expires"))
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
	assertExpiryEquals(s.T(), j.CreatedAt.Add(h.JobTimeout), s.rr.Header().Get("Expires"))

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

	j.UpdatedAt = time.Now().Add(-(h.JobTimeout + time.Second))
	postgrestest.UpdateJob(s.T(), s.db, j)

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	JobStatus(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	assertExpiryEquals(s.T(), j.UpdatedAt.Add(h.JobTimeout), s.rr.Header().Get("Expires"))
}

func (s *APITestSuite) TestDeleteJobBadInputs() {
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
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/jobs/%s", tt.jobID), nil)
			rr := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			ad := s.makeContextValues(acoUnderTest)
			req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

			JobStatus(rr, req)

			respOO := getOperationOutcome(t, rr.Body.Bytes())

			assert.Equal(t, fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
			assert.Equal(t, fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
			assert.Equal(t, tt.expErrCode, respOO.Issue[0].Details.Coding[0].Code.Value)
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
				RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
				Status:     tt.status,
			}
			postgrestest.CreateJobs(t, s.db, &j)
			defer postgrestest.DeleteJobByID(t, s.db, j.ID)

			req := s.createJobStatusRequest(acoUnderTest, j.ID)
			rr := httptest.NewRecorder()

			DeleteJob(rr, req)
			assert.Equal(t, tt.expStatusCode, rr.Code)
			if rr.Code == http.StatusGone {
				assert.Contains(t, rr.Body.String(), "Job was not cancelled because it is not Pending or In Progress")
			}
		})
	}
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

func getOperationOutcome(t *testing.T, data []byte) *fhirmodels.OperationOutcome {
	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", jsonformat.STU3)
	assert.NoError(t, err)
	container, err := unmarshaller.Unmarshal(data)
	assert.NoError(t, err)
	return container.(*fhirmodels.ContainedResource).GetOperationOutcome()
}
