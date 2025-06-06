package v3

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	appMiddleware "github.com/CMSgov/bcda-app/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirresources "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhiroo "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/operation_outcome_go_proto"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	expiryHeaderFormat = "2006-01-02 15:04:05.999999999 -0700 MST"
)

var (
	acoUnderTest = uuid.Parse(constants.LargeACOUUID)
)

type APITestSuite struct {
	suite.Suite
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
	})

	s.db = database.Connection

	// Set up the logger since we're using the real client
	client.SetLogger(log.BBAPI)
}

func (s *APITestSuite) TearDownTest() {
	postgrestest.DeleteJobsByACOID(s.T(), s.db, acoUnderTest)
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
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
			req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs/%s", constants.V3Path, tt.jobID), nil)
			rr := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			ad := s.makeContextValues(acoUnderTest)
			req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))

			JobStatus(rr, req)

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
				RequestURL: constants.V3Path + constants.PatientEOBPath,
				Status:     tt.status,
			}
			postgrestest.CreateJobs(t, s.db, &j)
			defer postgrestest.DeleteJobByID(t, s.db, j.ID)

			req := s.createJobStatusRequest(acoUnderTest, j.ID)
			rr := httptest.NewRecorder()

			JobStatus(rr, req)
			assert.Equal(t, tt.expStatusCode, rr.Code)
			switch rr.Code {
			case http.StatusAccepted:
				assert.Contains(t, rr.Header().Get("X-Progress"), tt.status)
				assert.Equal(t, "", rr.Header().Get("Expires"))
			case http.StatusInternalServerError:
				assert.Contains(t, rr.Body.String(), "Service encountered numerous errors")
			case http.StatusGone:
				assertExpiryEquals(t, j.CreatedAt.Add(h.JobTimeout), rr.Header().Get("Expires"))
			}
		})
	}
}

// TODO: V3
// Some of the JobStatus tests seem to rely on existing non-cleaned up state.
// They are failing when run as part of the whole unit-test suite however succeed when run independently.
// These tests also cause failures in api/v2 tests when run as part of the whole suite.

// https://stackoverflow.com/questions/34585957/postgresql-9-3-how-to-insert-upper-case-uuid-into-table
func (s *APITestSuite) TestJobStatusCompleted() {
	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: constants.V3Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	var expectedUrls []string
	for i := 1; i <= 10; i++ {
		fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
		expectedurl := fmt.Sprintf("%s/%s/%s", constants.ExpectedTestUrl, fmt.Sprint(j.ID), fileName)
		expectedUrls = append(expectedUrls, expectedurl)
		postgrestest.CreateJobKeys(s.T(), s.db,
			models.JobKey{JobID: j.ID, FileName: fileName, ResourceType: "ExplanationOfBenefit"})
	}

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	rr := httptest.NewRecorder()

	JobStatus(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Code)
	assert.Equal(s.T(), constants.JsonContentType, rr.Header().Get(constants.ContentType))
	str := rr.Header().Get("Expires")
	fmt.Println(str)
	assertExpiryEquals(s.T(), j.CreatedAt.Add(h.JobTimeout), rr.Header().Get("Expires"))

	var rb api.BulkResponseBody
	err := json.Unmarshal(rr.Body.Bytes(), &rb)
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
		RequestURL: constants.V3Path + constants.PatientEOBPath,
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
	rr := httptest.NewRecorder()

	JobStatus(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Code)
	assert.Equal(s.T(), constants.JsonContentType, rr.Header().Get(constants.ContentType))

	var rb api.BulkResponseBody
	err = json.Unmarshal(rr.Body.Bytes(), &rb)
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
		RequestURL: constants.V3Path + constants.PatientEOBPath,
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	j.UpdatedAt = time.Now().Add(-(h.JobTimeout + time.Second))
	postgrestest.UpdateJob(s.T(), s.db, j)

	req := s.createJobStatusRequest(acoUnderTest, j.ID)
	rr := httptest.NewRecorder()

	JobStatus(rr, req)

	assert.Equal(s.T(), http.StatusGone, rr.Code)
	assertExpiryEquals(s.T(), j.UpdatedAt.Add(h.JobTimeout), rr.Header().Get("Expires"))
}

func (s *APITestSuite) TestJobsStatus() {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs", constants.V3Path), nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: fmt.Sprintf("%sPatient/$export?_type=ExplanationOfBenefit", constants.V3Path),
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.db, j.ID)

	JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)
}

func (s *APITestSuite) TestJobsStatusNotFound() {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs", constants.V3Path), nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusNotFound, rr.Code)
}

func (s *APITestSuite) TestJobsStatusNotFoundWithStatus() {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs?_status=Failed", constants.V3Path), nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: fmt.Sprintf("%sPatient/$export?_type=ExplanationOfBenefit", constants.V3Path),
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.db, j.ID)

	JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusNotFound, rr.Code)
}

func (s *APITestSuite) TestJobsStatusWithStatus() {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs?_status=Failed", constants.V3Path), nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: fmt.Sprintf("%sPatient/$export?_type=ExplanationOfBenefit", constants.V3Path),
		Status:     models.JobStatusFailed,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.db, j.ID)

	JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)
}

func (s *APITestSuite) TestJobsStatusWithStatuses() {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs?_status=Completed,Failed", constants.V3Path), nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	j := models.Job{
		ACOID:      acoUnderTest,
		RequestURL: fmt.Sprintf("%sPatient/$export?_type=ExplanationOfBenefit", constants.V3Path),
		Status:     models.JobStatusFailed,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)
	defer postgrestest.DeleteJobByID(s.T(), s.db, j.ID)

	JobsStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)
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
			req := httptest.NewRequest("DELETE", fmt.Sprintf("%sjobs/%s", constants.V3Path, tt.jobID), nil)
			rr := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			ad := s.makeContextValues(acoUnderTest)
			req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
			JobStatus(rr, req)

			respOO := getOperationOutcome(t, rr.Body.Bytes())

			assert.Equal(t, fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
			assert.Equal(t, fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
			assert.Equal(t, tt.expErrCode, respOO.Issue[0].Diagnostics.Value)
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
				RequestURL: fmt.Sprintf("%sPatient/$export?_type=Patient,Coverage", constants.V3Path),
				Status:     tt.status,
			}
			postgrestest.CreateJobs(t, s.db, &j)
			defer postgrestest.DeleteJobByID(t, s.db, j.ID)

			req := s.createJobStatusRequest(acoUnderTest, j.ID)
			rr := httptest.NewRecorder()

			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))

			DeleteJob(rr, req)
			assert.Equal(t, tt.expStatusCode, rr.Code)
			if rr.Code == http.StatusGone {
				assert.Contains(t, rr.Body.String(), "job was not cancelled because it is not Pending or In Progress")
			}
		})
	}
}
func (s *APITestSuite) TestMetadataResponse() {
	ts := httptest.NewServer(http.HandlerFunc(Metadata))
	defer ts.Close()

	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	assert.NoError(s.T(), err)

	res, err := http.Get(ts.URL)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), "application/json", res.Header.Get(constants.ContentType))
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	resp, err := io.ReadAll(res.Body)
	assert.NoError(s.T(), err)

	resource, err := unmarshaller.Unmarshal(resp)
	assert.NoError(s.T(), err)
	cs := resource.(*fhirresources.ContainedResource).GetCapabilityStatement()

	// Expecting an R4 response so we'll evaluate some fields to reflect that
	assert.Equal(s.T(), fhircodes.FHIRVersionCode_V_4_0_1, cs.FhirVersion.Value)
	assert.Equal(s.T(), 1, len(cs.Rest))
	assert.Equal(s.T(), 2, len(cs.Rest[0].Resource))
	assert.Len(s.T(), cs.Instantiates, 2)
	assert.Contains(s.T(), cs.Instantiates[0].Value, fmt.Sprintf("%s/metadata", constants.BFDV3Path))
	resourceData := []struct {
		rt           fhircodes.ResourceTypeCode_Value
		opName       string
		opDefinition string
	}{
		{fhircodes.ResourceTypeCode_PATIENT, "patient-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
		{fhircodes.ResourceTypeCode_GROUP, "group-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
	}

	for _, rd := range resourceData {
		for _, r := range cs.Rest[0].Resource {
			if r.Type.Value == rd.rt {
				assert.NotNil(s.T(), r)
				assert.Equal(s.T(), 1, len(r.Operation))
				assert.Equal(s.T(), rd.opName, r.Operation[0].Name.Value)
				assert.Equal(s.T(), rd.opDefinition, r.Operation[0].Definition.Value)
				break
			}
		}
	}

	extensions := cs.Rest[0].Security.Extension
	assert.Len(s.T(), extensions, 1)
	assert.Equal(s.T(), "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris", extensions[0].Url.Value)

	subExtensions := extensions[0].Extension
	assert.Len(s.T(), subExtensions, 1)
	assert.Equal(s.T(), "token", subExtensions[0].Url.Value)
	assert.Equal(s.T(), ts.URL+"/auth/token", subExtensions[0].GetValue().GetUri().Value)

}

func (s *APITestSuite) TestResourceTypes() {
	tests := []struct {
		name          string
		resourceTypes []string
		statusCode    int
	}{
		{"Supported type - Patient", []string{"Patient"}, http.StatusAccepted},
		{"Supported type - Coverage", []string{"Coverage"}, http.StatusAccepted},
		{"Supported type - Patient,Coverage", []string{"Patient", "Coverage"}, http.StatusAccepted},
		{"Supported type - EOB", []string{"ExplanationOfBenefit"}, http.StatusAccepted},
		{"Supported type - default", nil, http.StatusAccepted},
	}

	resources, _ := service.GetDataTypes([]string{
		"Patient",
		"Coverage",
		"ExplanationOfBenefit",
	}...)

	h := api.NewHandler(resources, constants.BFDV3Path, constants.V3Version)
	mockSvc := &service.MockService{}

	mockSvc.On("GetLatestCCLFFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.CCLFFile{PerformanceYear: utils.GetPY()}, nil)
	mockAco := service.ACOConfig{
		Data: []string{"adjudicated"},
	}
	mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockAco, true)
	mockSvc.On("GetTimeConstraints", testUtils.CtxMatcher, mock.AnythingOfType("string")).Return(service.TimeConstraints{}, nil)
	mockSvc.On("GetCutoffTime", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Time{}, constants.GetExistingBenes)
	h.Svc = mockSvc

	for idx, handler := range []http.HandlerFunc{h.BulkGroupRequest, h.BulkPatientRequest} {
		for _, tt := range tests {
			s.T().Run(fmt.Sprintf("%s-%d", tt.name, idx), func(t *testing.T) {
				rr := httptest.NewRecorder()

				ep := "Group"
				if idx == 1 {
					ep = "Patient"
				}

				u, err := url.Parse(fmt.Sprintf("%s%s/$export", constants.V3Path, ep))
				assert.NoError(t, err)

				rp := middleware.RequestParameters{
					Version:       constants.V3Version,
					ResourceTypes: tt.resourceTypes,
				}

				req := httptest.NewRequest("GET", u.String(), nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("groupId", "all")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
				req = req.WithContext(context.WithValue(req.Context(), appMiddleware.CtxTransactionKey, uuid.New()))
				ad := s.getAuthData()
				req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
				req = req.WithContext(middleware.SetRequestParamsCtx(req.Context(), rp))
				newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
				req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))

				handler(rr, req)
				assert.Equal(t, tt.statusCode, rr.Code)
				assert.Empty(t, rr.Body.String())
				if rr.Code == http.StatusAccepted {
					assert.NotEmpty(t, rr.Header().Get("Content-Location"))
				}
			})
		}
	}
}

func (s *APITestSuite) TestGetAttributionStatus() {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sattribution_status", constants.V3Path), nil)
	ad := s.makeContextValues(acoUnderTest)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	rr := httptest.NewRecorder()

	AttributionStatus(rr, req)
	assert.Equal(s.T(), http.StatusOK, rr.Code)

	var resp api.AttributionFileStatusResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)

	aco := postgrestest.GetACOByUUID(s.T(), s.db, acoUnderTest)
	cclfFile := postgrestest.GetLatestCCLFFileByCMSIDAndType(s.T(), s.db, *aco.CMSID, models.FileTypeDefault)

	assert.Equal(s.T(), "last_attribution_update", resp.Data[0].Type)
	assert.Equal(s.T(), cclfFile.Timestamp.Format("2006-01-02 15:04:05"), resp.Data[0].Timestamp.Format("2006-01-02 15:04:05"))
}

func (s *APITestSuite) getAuthData() (data auth.AuthData) {
	aco := postgrestest.GetACOByUUID(s.T(), s.db, acoUnderTest)
	return auth.AuthData{ACOID: acoUnderTest.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
}

func (s *APITestSuite) makeContextValues(acoID uuid.UUID) (data auth.AuthData) {
	aco := postgrestest.GetACOByUUID(s.T(), s.db, acoID)
	return auth.AuthData{ACOID: aco.UUID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
}

func (s *APITestSuite) createJobStatusRequest(acoID uuid.UUID, jobID uint) *http.Request {
	req := httptest.NewRequest("GET", fmt.Sprintf("%sjobs/%d", constants.V3Path, jobID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(jobID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := s.makeContextValues(acoID)
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
	return req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
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

func getOperationOutcome(t *testing.T, data []byte) *fhiroo.OperationOutcome {
	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	assert.NoError(t, err)
	container, err := unmarshaller.Unmarshal(data)
	assert.NoError(t, err)
	return container.(*fhirresources.ContainedResource).GetOperationOutcome()
}

func MakeTestStructuredLoggerEntry(logFields logrus.Fields) *log.StructuredLoggerEntry {
	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logFields)}
	return newLogEntry
}
