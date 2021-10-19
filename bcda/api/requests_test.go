package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"

	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/google/fhir/go/jsonformat"
	fhircodesv2 "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirmodelv2CR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhircodesv1 "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirmodelsv1 "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

type RequestsTestSuite struct {
	suite.Suite

	runoutEnabledEnvVar string

	db *sql.DB

	acoID uuid.UUID

	resourceType map[string]service.DataType
}

func TestRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(RequestsTestSuite))
}

func (s *RequestsTestSuite) SetupSuite() {
	// See testdata/acos.yml
	s.acoID = uuid.Parse("ba21d24d-cd96-4d7d-a691-b0e8c88e67a5")
	s.db, _ = databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	tf, err := testfixtures.New(
		testfixtures.Database(s.db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)

	s.resourceType = map[string]service.DataType{
		"Patient":              {Adjudicated: true},
		"Coverage":             {Adjudicated: true},
		"ExplanationOfBenefit": {Adjudicated: true},
	}

	if err != nil {
		assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
	}
	if err := tf.Load(); err != nil {
		assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
	}

	// Set up the logger since we're using the real client
	client.SetLogger(log.BBAPI)
}

func (s *RequestsTestSuite) SetupTest() {
	s.runoutEnabledEnvVar = conf.GetEnv("BCDA_ENABLE_RUNOUT")
}

func (s *RequestsTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", s.runoutEnabledEnvVar)
}

func (s *RequestsTestSuite) TestRunoutEnabled() {
	conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", "true")
	qj := []*models.JobEnqueueArgs{}
	tests := []struct {
		name string

		errToReturn error
		respCode    int
		apiVersion  string
	}{
		{"Successful", nil, http.StatusAccepted, "v1"},
		{"Successful v2", nil, http.StatusAccepted, "v2"},
		{"No CCLF file found", service.CCLFNotFoundError{}, http.StatusNotFound, "v1"},
		{"No CCLF file found v2", service.CCLFNotFoundError{}, http.StatusNotFound, "v2"},
		{"Some other error", errors.New("Some other error"), http.StatusInternalServerError, "v1"},
		{"Some other error v2", errors.New("Some other error"), http.StatusInternalServerError, "v2"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &service.MockService{}
			var jobs []*models.JobEnqueueArgs
			if tt.errToReturn == nil {
				jobs = qj
			}

			resourceMap := s.resourceType

			mockSvc.On("GetQueJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(jobs, tt.errToReturn)
			mockAco := service.ACOConfig{Data: []string{"adjudicated"}}
			mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockAco, true)
			h := newHandler(resourceMap, fmt.Sprintf("/%s/fhir", tt.apiVersion), tt.apiVersion, s.db)
			h.Svc = mockSvc

			req := s.genGroupRequest("runout", middleware.RequestParameters{})
			w := httptest.NewRecorder()
			h.BulkGroupRequest(w, req)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.NoError(t, err)
			assert.Equal(t, tt.respCode, resp.StatusCode)
			if tt.errToReturn == nil {
				assert.NotEmpty(t, resp.Header.Get("Content-Location"))
			} else {
				assert.Contains(t, string(body), tt.errToReturn.Error())
			}
		})
	}
}

func (s *RequestsTestSuite) TestJobsStatusV1() {
	apiVersion := "v1"

	tests := []struct {
		name string

		respCode int
		statuses []models.JobStatus
		codes    []fhircodesv1.TaskStatusCode_Value
	}{
		{"Successful with no status(es)", http.StatusOK, nil, []fhircodesv1.TaskStatusCode_Value{fhircodesv1.TaskStatusCode_COMPLETED}},
		{"Successful with one status", http.StatusOK, []models.JobStatus{models.JobStatusCompleted}, []fhircodesv1.TaskStatusCode_Value{fhircodesv1.TaskStatusCode_COMPLETED}},
		{"Successful with two statuses", http.StatusOK, []models.JobStatus{models.JobStatusCompleted, models.JobStatusFailed}, []fhircodesv1.TaskStatusCode_Value{fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_FAILED}},
		{"Successful with all statuses", http.StatusOK, models.AllJobStatuses,
			[]fhircodesv1.TaskStatusCode_Value{
				fhircodesv1.TaskStatusCode_IN_PROGRESS, fhircodesv1.TaskStatusCode_IN_PROGRESS, fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_FAILED, fhircodesv1.TaskStatusCode_CANCELLED, fhircodesv1.TaskStatusCode_FAILED, fhircodesv1.TaskStatusCode_CANCELLED,
			},
		},
		{"Jobs not found", http.StatusNotFound, []models.JobStatus{models.JobStatusCompleted}, nil},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &service.MockService{}

			switch tt.respCode {
			case http.StatusNotFound:
				mockSvc.On("GetJobs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(
					nil, service.JobsNotFoundError{},
				)
			case http.StatusOK:
				var (
					jobs     []*models.Job
					mockArgs []interface{}
				)

				mockArgs = append(mockArgs, testUtils.CtxMatcher, mock.Anything)
				if tt.statuses == nil {
					jobs = s.addNewJob(jobs, uint(1), models.JobStatusCompleted, apiVersion)
					for range models.AllJobStatuses {
						mockArgs = append(mockArgs, mock.Anything)
					}
				} else {
					for k := range tt.statuses {
						mockArgs = append(mockArgs, mock.Anything)
						jobs = s.addNewJob(jobs, uint(k), tt.statuses[k], apiVersion)
					}
				}

				mockSvc.On("GetJobs", mockArgs...).Return(
					jobs, nil,
				)
			}

			h := newHandler(map[string]service.DataType{
				"Patient":              {},
				"Coverage":             {},
				"ExplanationOfBenefit": {},
			}, "/v1/fhir", "v1", s.db)
			h.Svc = mockSvc

			rr := httptest.NewRecorder()
			req := s.genGetJobsRequest("v1", tt.statuses)
			h.JobsStatus(rr, req)

			unmarshaller, err := jsonformat.NewUnmarshaller("UTC", jsonformat.STU3)
			assert.NoError(s.T(), err)

			switch tt.respCode {
			case http.StatusNotFound:
				assert.Equal(s.T(), http.StatusNotFound, rr.Code)
			case http.StatusOK:
				assert.Equal(s.T(), tt.respCode, rr.Result().StatusCode)

				resp, err := unmarshaller.Unmarshal(rr.Body.Bytes())
				assert.NoError(s.T(), err)

				bundle := resp.(*fhirmodelsv1.ContainedResource)
				respB := bundle.GetBundle()
				assert.Equal(s.T(), http.StatusOK, rr.Code)
				assert.Equal(s.T(), uint32(len(respB.Entry)), respB.Total.Value)

				for k, entry := range respB.Entry {
					respT := entry.GetResource().GetTask()
					assert.Equal(s.T(), respT.Status.Value, tt.codes[k])
					assert.Equal(s.T(), respT.Input[0].Value.GetStringValue().Value, "GET https://bcda.test.gov/v1/this-is-a-test")
				}
			}
		})
	}
}

func (s *RequestsTestSuite) TestJobsStatusV2() {
	apiVersion := "v2"

	tests := []struct {
		name string

		respCode int
		statuses []models.JobStatus
		codes    []fhircodesv2.TaskStatusCode_Value
	}{
		{"Successful with no status(es)", http.StatusOK, nil, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED}},
		{"Successful with one status", http.StatusOK, []models.JobStatus{models.JobStatusCompleted}, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED}},
		{"Successful with two statuses", http.StatusOK, []models.JobStatus{models.JobStatusCompleted, models.JobStatusFailed}, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_FAILED}},
		{"Successful with all statuses", http.StatusOK, models.AllJobStatuses,
			[]fhircodesv2.TaskStatusCode_Value{
				fhircodesv2.TaskStatusCode_IN_PROGRESS, fhircodesv2.TaskStatusCode_IN_PROGRESS, fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_FAILED, fhircodesv2.TaskStatusCode_CANCELLED, fhircodesv2.TaskStatusCode_FAILED, fhircodesv2.TaskStatusCode_CANCELLED,
			},
		},
		{"Jobs not found", http.StatusNotFound, []models.JobStatus{models.JobStatusCompleted}, nil},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &service.MockService{}

			switch tt.respCode {
			case http.StatusNotFound:
				mockSvc.On("GetJobs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(
					nil, service.JobsNotFoundError{},
				)
			case http.StatusOK:
				var (
					jobs     []*models.Job
					mockArgs []interface{}
				)

				mockArgs = append(mockArgs, testUtils.CtxMatcher, mock.Anything)
				if tt.statuses == nil {
					jobs = s.addNewJob(jobs, uint(1), models.JobStatusCompleted, apiVersion)
					for range models.AllJobStatuses {
						mockArgs = append(mockArgs, mock.Anything)
					}
				} else {
					for k := range tt.statuses {
						mockArgs = append(mockArgs, mock.Anything)
						jobs = s.addNewJob(jobs, uint(k), tt.statuses[k], apiVersion)
					}
				}

				mockSvc.On("GetJobs", mockArgs...).Return(
					jobs, nil,
				)
			}

			h := newHandler(map[string]service.DataType{
				"Patient":              {},
				"Coverage":             {},
				"ExplanationOfBenefit": {},
			}, "/v2/fhir", "v2", s.db)
			h.Svc = mockSvc

			rr := httptest.NewRecorder()
			req := s.genGetJobsRequest("v2", tt.statuses)
			h.JobsStatus(rr, req)

			unmarshaller, err := jsonformat.NewUnmarshaller("UTC", jsonformat.R4)
			assert.NoError(s.T(), err)

			switch tt.respCode {
			case http.StatusNotFound:
				assert.Equal(s.T(), http.StatusNotFound, rr.Code)
			case http.StatusOK:
				assert.Equal(s.T(), tt.respCode, rr.Result().StatusCode)

				resp, err := unmarshaller.Unmarshal(rr.Body.Bytes())
				assert.NoError(s.T(), err)

				bundle := resp.(*fhirmodelv2CR.ContainedResource)
				respB := bundle.GetBundle()
				assert.Equal(s.T(), http.StatusOK, rr.Code)
				assert.Equal(s.T(), uint32(len(respB.Entry)), respB.Total.Value)

				for k, entry := range respB.Entry {
					respT := entry.GetResource().GetTask()
					assert.Equal(s.T(), respT.Status.Value, tt.codes[k])
					assert.Equal(s.T(), respT.Input[0].Value.GetStringValue().Value, "GET https://bcda.test.gov/v2/this-is-a-test")
				}
			}
		})
	}
}

func (s *RequestsTestSuite) addNewJob(jobs []*models.Job, id uint, status models.JobStatus, apiVersion string) []*models.Job {
	return append(jobs, &models.Job{
		ID:         id,
		ACOID:      uuid.NewUUID(),
		Status:     status,
		RequestURL: "https://bcda.test.gov/" + apiVersion + "/this-is-a-test",
		CreatedAt:  time.Now().Add(-24 * time.Hour),
		UpdatedAt:  time.Now(),
	})
}

func (s *RequestsTestSuite) TestAttributionStatus() {
	tests := []struct {
		name string

		respCode  int
		fileNames []string
		fileTypes []string
	}{
		{"Successful with both files", http.StatusOK, []string{"cclf_test_file_1", "cclf_test_file_2"}, []string{"last_attribution_update", "last_runout_update"}},
		{"Successful with default file", http.StatusOK, []string{"cclf_test_file_1", ""}, []string{"last_attribution_update", ""}},
		{"Successful with runout file", http.StatusOK, []string{"", "cclf_test_file_2"}, []string{"", "last_runout_update"}},
		{"No CCLF files found", http.StatusNotFound, []string{"", ""}, []string{"", ""}},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &service.MockService{}

			for i, name := range tt.fileNames {
				fileType := models.FileTypeDefault
				if i == 1 {
					fileType = models.FileTypeRunout
				}
				switch name {
				case "":
					mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, fileType).Return(
						nil,
						service.CCLFNotFoundError{
							FileNumber: 8,
							CMSID:      "",
							FileType:   0,
							CutoffTime: time.Time{}},
					)
				default:
					mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, fileType).Return(
						&models.CCLFFile{
							ID:        1,
							Name:      tt.fileNames[i],
							Timestamp: time.Time{},
							CCLFNum:   8,
						},
						nil,
					)
				}
			}

			resourceMap := s.resourceType
			h := newHandler(resourceMap, "/v1/fhir", "v1", s.db)
			h.Svc = mockSvc

			rr := httptest.NewRecorder()
			req := s.genASRequest()
			h.AttributionStatus(rr, req)

			switch tt.respCode {
			case http.StatusNotFound:
				assert.Equal(s.T(), http.StatusNotFound, rr.Code)
			case http.StatusOK:
				var resp AttributionFileStatusResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(s.T(), err)

				count := 0
				for _, fileStatus := range resp.Data {
					if tt.fileNames[count] != "" {
						assert.Equal(s.T(), tt.fileTypes[count], fileStatus.Type)
						count += 1
					}
				}
			}
		})
	}
}

func (s *RequestsTestSuite) TestRunoutDisabled() {
	conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", "false")
	req := s.genGroupRequest("runout", middleware.RequestParameters{})
	w := httptest.NewRecorder()
	h := &Handler{}
	h.RespWriter = responseutils.NewResponseWriter()
	h.BulkGroupRequest(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Contains(string(body), "Invalid group ID")
}

func (s *RequestsTestSuite) TestDataTypeAuthorization() {
	acoA := &service.ACOConfig{
		Model:              "Model A",
		Pattern:            "A\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{"adjudicated", "pre-adjudicated"},
	}
	acoB := &service.ACOConfig{
		Model:              "Model B",
		Pattern:            "B\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{"adjudicated"},
	}
	acoC := &service.ACOConfig{
		Model:              "Model C",
		Pattern:            "C\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{"pre-adjudicated"},
	}
	acoD := &service.ACOConfig{
		Model:              "Model D",
		Pattern:            "D\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{},
	}

	dataTypeMap := map[string]service.DataType{
		"Coverage":             {Adjudicated: true},
		"Patient":              {Adjudicated: true},
		"ExplanationOfBenefit": {Adjudicated: true},
		"Claim":                {Adjudicated: false, PreAdjudicated: true},
		"ClaimResponse":        {Adjudicated: false, PreAdjudicated: true},
	}

	h := NewHandler(dataTypeMap, "/v2/fhir/", "v2")

	// Use a mock to ensure that this test does not generate artifacts in the queue for other tests
	mockEnq := &queueing.MockEnqueuer{}
	mockEnq.On("AddJob", mock.Anything, mock.Anything).Return(nil)
	h.Enq = mockEnq
	h.supportedDataTypes = dataTypeMap

	client.SetLogger(log.API) // Set logger so we don't get errors later

	jsonBytes, _ := json.Marshal("{}")

	tests := []struct {
		name string

		cmsId        string
		resources    []string
		expectedCode int
		acoConfig    *service.ACOConfig
	}{
		{"Auth Adj/Pre-Adj, Request Adj/Pre-Adj", "A0000", []string{"Claim", "Patient"}, http.StatusAccepted, acoA},
		{"Auth Adj, Request Adj", "B0000", []string{"Patient"}, http.StatusAccepted, acoB},
		{"Auth Adj, Request Pre-Adj", "B0000", []string{"Claim"}, http.StatusBadRequest, acoB},
		{"Auth Pre-Adj, Request Adj", "C0000", []string{"Patient"}, http.StatusBadRequest, acoC},
		{"Auth Pre-Adj, Request Pre-Adj", "C0000", []string{"Claim"}, http.StatusAccepted, acoC},
		{"Auth None, Request Adj", "D0000", []string{"Patient"}, http.StatusBadRequest, acoD},
		{"Auth None, Request Pre-Adj", "D0000", []string{"Claim"}, http.StatusBadRequest, acoD},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(t *testing.T) {
			mockSvc := service.MockService{}

			mockSvc.On("GetQueJobs", mock.Anything, mock.Anything).Return([]*models.JobEnqueueArgs{}, nil)
			mockSvc.On("GetACOConfigForID", mock.Anything).Return(test.acoConfig, true)

			h.Svc = &mockSvc

			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "http://bcda.ms.gov/api/v2/Group/$export", bytes.NewReader(jsonBytes))

			r = r.WithContext(context.WithValue(r.Context(), auth.AuthDataContextKey, auth.AuthData{
				ACOID: "8d80925a-027e-43dd-8aed-9a501cc4cd91",
				CMSID: test.cmsId,
			}))

			r = r.WithContext(middleware.NewRequestParametersContext(r.Context(), middleware.RequestParameters{
				Since:         time.Date(2000, 01, 01, 00, 00, 00, 00, time.UTC),
				ResourceTypes: test.resources,
				Version:       "v2",
			}))

			h.bulkRequest(w, r, service.DefaultRequest)

			assert.Equal(s.T(), test.expectedCode, w.Code)
		})
	}
}

// TestRequests verifies that we can initiate an export job for all resource types using all the different handlers
func (s *RequestsTestSuite) TestRequests() {

	resourceMap := s.resourceType

	h := newHandler(resourceMap, "/v1/fhir", "v1", s.db)

	// Use a mock to ensure that this test does not generate artifacts in the queue for other tests
	enqueuer := &queueing.MockEnqueuer{}
	enqueuer.On("AddJob", mock.Anything, mock.Anything).Return(nil)
	h.Enq = enqueuer
	mockSvc := service.MockService{}

	mockSvc.On("GetQueJobs", mock.Anything, mock.Anything).Return([]*models.JobEnqueueArgs{}, nil)
	mockAco := service.ACOConfig{
		Data: []string{"adjudicated"},
	}
	mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything).Return(&mockAco, true)

	h.Svc = &mockSvc

	// Test Group and Patient
	// Patient, Coverage, and ExplanationOfBenefit
	// with And without Since parameter
	resources := []string{"Patient", "ExplanationOfBenefit", "Coverage"}
	sinces := []time.Time{{}, time.Now().Round(time.Millisecond).Add(-24 * time.Hour)}
	groupIDs := []string{"all", "runout"}

	// Validate group requests
	for _, resource := range resources {
		for _, since := range sinces {
			for _, groupID := range groupIDs {
				rp := middleware.RequestParameters{
					Version:       "v1",
					ResourceTypes: []string{resource},
					Since:         since,
				}
				rr := httptest.NewRecorder()
				req := s.genGroupRequest(groupID, rp)
				h.BulkGroupRequest(rr, req)
				assert.Equal(s.T(), http.StatusAccepted, rr.Code)
			}
		}
	}

	// Validate patient requests
	for _, resource := range resources {
		for _, since := range sinces {
			rp := middleware.RequestParameters{
				Version:       "v1",
				ResourceTypes: []string{resource},
				Since:         since,
			}
			rr := httptest.NewRecorder()
			h.BulkPatientRequest(rr, s.genPatientRequest(rp))
			assert.Equal(s.T(), http.StatusAccepted, rr.Code)
		}
	}
}

func (s *RequestsTestSuite) TestJobStatus() {
	resourceMap := s.resourceType
	h := newHandler(resourceMap, "/v1/fhir", "v1", s.db)
	mockSrv := service.MockService{}
	timestp := time.Now()
	mockSrv.On("GetJobAndKeys", testUtils.CtxMatcher, uint(1)).Return(
		&models.Job{
			ID:                1,
			ACOID:             uuid.NewRandom(),
			RequestURL:        "http://bcda.cms.gov/api/v1/Jobs/1",
			Status:            models.JobStatusCompleted,
			TransactionTime:   timestp,
			JobCount:          100,
			CompletedJobCount: 100,
			CreatedAt:         timestp,
			UpdatedAt:         timestp,
		},
		[]*models.JobKey{{
			ID:           1,
			JobID:        1,
			FileName:     "testingtesting",
			ResourceType: "Patient",
		}},
		nil,
	)
	h.Svc = &mockSrv

	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/jobs/1", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "1")

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.JobStatus(w, req)
	s.Equal(http.StatusOK, w.Code)
}

func (s *RequestsTestSuite) genGroupRequest(groupID string, rp middleware.RequestParameters) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/Group/$export", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", groupID)

	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)
	ctx = middleware.NewRequestParametersContext(ctx, rp)

	req = req.WithContext(ctx)

	return req
}

func (s *RequestsTestSuite) genPatientRequest(rp middleware.RequestParameters) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/Patient/$export", nil)
	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
	ctx = middleware.NewRequestParametersContext(ctx, rp)

	return req.WithContext(ctx)
}

func (s *RequestsTestSuite) genASRequest() *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/attribution_status", nil)
	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)

	return req.WithContext(ctx)
}

func (s *RequestsTestSuite) genGetJobsRequest(version string, statuses []models.JobStatus) *http.Request {
	target := fmt.Sprintf("http://bcda.cms.gov/api/%s/jobs", version)
	if statuses != nil {
		target = target + "?_status="
		for _, status := range statuses {
			target = target + string(status) + ","
		}
		target = strings.TrimRight(target, ",")
	}
	target = strings.ReplaceAll(target, " ", "%20") // Remove possible spaces in query parameter

	req := httptest.NewRequest("GET", target, nil)

	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)

	return req.WithContext(ctx)
}
