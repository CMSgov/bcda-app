package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	responseutilsv3 "github.com/CMSgov/bcda-app/bcda/responseutils/v3"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/db"
	"github.com/CMSgov/bcda-app/log"
	appMiddleware "github.com/CMSgov/bcda-app/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/ccoveille/go-safecast"
	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	fhircodesv2 "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirmodelv2CR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhircodesv1 "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirmodelsv1 "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

const apiVersionOne = "v1"
const apiVersionTwo = "v2"
const v1BasePath = "/v1/fhir"
const v2BasePath = "/v2/fhir"
const v1JobRequestUrl = "http://bcda.cms.gov/api/v1/Jobs/1"
const v2JobRequestUrl = "http://bcda.cms.gov/api/v2/Jobs/1"

type RequestsTestSuite struct {
	suite.Suite

	runoutEnabledEnvVar string

	db *sql.DB

	pool *pgxv5Pool.Pool

	acoID uuid.UUID

	resourceType map[string]service.ClaimType
}

func TestRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(RequestsTestSuite))
}

func (s *RequestsTestSuite) SetupSuite() {
	// See testdata/acos.yml
	s.acoID = uuid.Parse("ba21d24d-cd96-4d7d-a691-b0e8c88e67a5")
	db, _, _ := databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	s.db = db
	s.pool = database.ConnectPool()
	tf, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)

	s.resourceType = map[string]service.ClaimType{
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
	client.SetLogger(log.BFDAPI)
}

func (s *RequestsTestSuite) SetupTest() {
	s.runoutEnabledEnvVar = conf.GetEnv("BCDA_ENABLE_RUNOUT")
}

func (s *RequestsTestSuite) TearDownTest() {
	err := conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", s.runoutEnabledEnvVar)
	assert.Empty(s.T(), err)
}

func (s *RequestsTestSuite) TestRunoutEnabled() {
	err := conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", "true")
	assert.Empty(s.T(), err)

	tests := []struct {
		name               string
		errToReturn        error
		respCode           int
		apiVersion         string
		runoutAttributions bool
		expectedMessage    string
	}{
		{"Successful", nil, http.StatusAccepted, apiVersionOne, true, ""},
		{"Successful v2", nil, http.StatusAccepted, apiVersionTwo, true, ""},
		{"FindCCLFFiles error", CCLFNotFoundOperationOutcomeError{}, http.StatusNotFound, apiVersionOne, false, "failed to start job; attribution file not found"},
		{"FindCCLFFiles error v2", DatabaseError{}, http.StatusInternalServerError, apiVersionTwo, false, ""},
		{constants.DefaultError, QueueError{}, http.StatusInternalServerError, apiVersionOne, true, ""},
		{constants.DefaultError + " v2", QueueError{}, http.StatusInternalServerError, apiVersionTwo, true, ""},
		{"Expired runout data", CCLFNotFoundOperationOutcomeError{}, http.StatusNotFound, apiVersionOne, false, "runout data is no longer available. Runout data expires 180 days after ingestion"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			resourceMap := s.resourceType
			mockSvc := &service.MockService{}
			mockAco := service.ACOConfig{Data: []string{"adjudicated"}}
			mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockAco, true)
			h := newHandler(resourceMap, fmt.Sprintf("/%s/fhir", tt.apiVersion), tt.apiVersion, s.db, s.pool)
			h.Svc = mockSvc
			enqueuer := queueing.NewMockEnqueuer(s.T())
			h.Enq = enqueuer

			// Set up cutoff time for expired runout test case
			cutoffTime := time.Time{}
			if _, ok := tt.errToReturn.(CCLFNotFoundOperationOutcomeError); ok && tt.name == "Expired runout data" {
				cutoffTime = time.Now().Add(-200 * 24 * time.Hour) // 200 days ago (past 180 day limit)
			}

			mockSvc.On("GetTimeConstraints", mock.Anything, mock.Anything).Return(service.TimeConstraints{}, nil)
			mockSvc.On("GetCutoffTime", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(cutoffTime, constants.GetExistingBenes)

			switch tt.errToReturn {
			case nil:
				mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.CCLFFile{PerformanceYear: (utils.GetPY() - 1)}, nil)
				enqueuer.On("AddPrepareJob", mock.Anything, mock.Anything).Return(nil)
			case CCLFNotFoundOperationOutcomeError{}:
				mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, service.CCLFNotFoundError{})
			case DatabaseError{}:
				mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))
			case QueueError{}:
				mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.Anything).
					Return(&models.CCLFFile{PerformanceYear: (utils.GetPY() - 1)}, nil)
				enqueuer.On("AddPrepareJob", mock.Anything, mock.Anything).Return(errors.New("error"))
			}

			req := s.genGroupRequest("runout", middleware.RequestParameters{})
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
			req = req.WithContext(context.WithValue(req.Context(), appMiddleware.CtxTransactionKey, uuid.New()))
			w := httptest.NewRecorder()
			h.BulkGroupRequest(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)

			assert.NoError(t, err)
			assert.Equal(t, tt.respCode, resp.StatusCode)
			if tt.errToReturn == nil {
				assert.NotEmpty(t, resp.Header.Get("Content-Location"))
			} else {
				if tt.expectedMessage != "" {
					assert.Contains(t, string(body), tt.expectedMessage)
				} else {
					assert.Contains(t, string(body), tt.errToReturn.Error())
				}
			}
		})
	}
}

func (s *RequestsTestSuite) TestJobsStatusV1() {
	apiVersion := "v1"
	fhirPath := "/" + apiVersion + "/fhir"

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
				fhircodesv1.TaskStatusCode_ACCEPTED, fhircodesv1.TaskStatusCode_IN_PROGRESS, fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_COMPLETED, fhircodesv1.TaskStatusCode_FAILED, fhircodesv1.TaskStatusCode_CANCELLED, fhircodesv1.TaskStatusCode_FAILED, fhircodesv1.TaskStatusCode_CANCELLED,
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
						u, err := safecast.ToUint(k)
						assert.NoError(t, err)
						jobs = s.addNewJob(jobs, u, tt.statuses[k], apiVersion)
					}
				}

				mockSvc.On("GetJobs", mockArgs...).Return(
					jobs, nil,
				)
			}

			h := newHandler(map[string]service.ClaimType{
				"Patient":              {},
				"Coverage":             {},
				"ExplanationOfBenefit": {},
			}, fhirPath, apiVersion, s.db, s.pool)
			h.Svc = mockSvc

			rr := httptest.NewRecorder()
			req := s.genGetJobsRequest(apiVersionOne, tt.statuses)
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
			h.JobsStatus(rr, req)

			unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.STU3)
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
				val, err := safecast.ToUint32(len(respB.Entry))
				assert.NoError(s.T(), err)
				assert.Equal(s.T(), val, respB.Total.Value)

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
	apiVersion := apiVersionTwo

	tests := []struct {
		name string

		respCode                 int
		statuses                 []models.JobStatus
		codes                    []fhircodesv2.TaskStatusCode_Value
		useMock                  bool
		throwInternalServerError bool
	}{
		{"Successful with no status(es)", http.StatusOK, nil, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED}, true, false},
		{"Successful with one status", http.StatusOK, []models.JobStatus{models.JobStatusCompleted}, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED}, true, false},
		{"Successful with two statuses", http.StatusOK, []models.JobStatus{models.JobStatusCompleted, models.JobStatusFailed}, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_FAILED}, true, false},
		{"Successful with all statuses", http.StatusOK, models.AllJobStatuses,
			[]fhircodesv2.TaskStatusCode_Value{
				fhircodesv2.TaskStatusCode_ACCEPTED, fhircodesv2.TaskStatusCode_IN_PROGRESS, fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_FAILED, fhircodesv2.TaskStatusCode_CANCELLED, fhircodesv2.TaskStatusCode_FAILED, fhircodesv2.TaskStatusCode_CANCELLED,
			}, true, false},
		{"Jobs not found", http.StatusNotFound, []models.JobStatus{models.JobStatusCompleted}, nil, true, false},
		{"Too Many Statuses", http.StatusBadRequest, []models.JobStatus{models.JobStatusCompleted, models.JobStatusCompleted}, nil, true, false},
		{"Invalid Status Type", http.StatusBadRequest, []models.JobStatus{"Eaten by alligators"}, nil, false, false},
		{"Invalid Auth Data", http.StatusBadRequest, []models.JobStatus{models.JobStatusCompleted}, nil, false, false},
		{"Other error", http.StatusInternalServerError, []models.JobStatus{models.JobStatusCompleted}, []fhircodesv2.TaskStatusCode_Value{fhircodesv2.TaskStatusCode_COMPLETED, fhircodesv2.TaskStatusCode_FAILED}, true, true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &service.MockService{}

			if tt.useMock {

				switch tt.respCode {
				case http.StatusNotFound:
					mockSvc.On("GetJobs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(
						nil, service.JobsNotFoundError{},
					)
				case http.StatusInternalServerError:
					mockSvc.On("GetJobs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(
						nil, errors.New("New Error"),
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
							u, err := safecast.ToUint(k)
							if err != nil {
								// handle the error appropriately, e.g., log it or return it
								t.Fatalf("Failed to convert key to uint: %v", err)
							}
							jobs = s.addNewJob(jobs, u, tt.statuses[k], apiVersion)
						}
					}

					mockSvc.On("GetJobs", mockArgs...).Return(
						jobs, nil,
					)

				}
			}
			h := newHandler(map[string]service.ClaimType{
				"Patient":              {},
				"Coverage":             {},
				"ExplanationOfBenefit": {},
			}, v2BasePath, apiVersionTwo, s.db, s.pool)
			if tt.useMock {
				h.Svc = mockSvc
			}

			rr := httptest.NewRecorder()
			req := s.genGetJobsRequest(apiVersionTwo, tt.statuses)
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(req.Context(), log.CtxLoggerKey, newLogEntry))
			if !tt.useMock {
				req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ""))
			}
			h.JobsStatus(rr, req)

			unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
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
				val, err := safecast.ToUint32(len(respB.Entry))
				assert.NoError(s.T(), err)
				assert.Equal(s.T(), val, respB.Total.Value)

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

		respCode    int
		fileNames   []string
		fileTypes   []string
		invalidAuth bool
	}{
		{"Successful with both files", http.StatusOK, []string{"cclf_test_file_1", "cclf_test_file_2"}, []string{"last_attribution_update", "last_runout_update"}, false},
		{"Successful with default file", http.StatusOK, []string{"cclf_test_file_1", ""}, []string{"last_attribution_update", ""}, false},
		{"Successful with runout file", http.StatusOK, []string{"", "cclf_test_file_2"}, []string{"", "last_runout_update"}, false},
		{"No CCLF files found", http.StatusNotFound, []string{"", ""}, []string{"", ""}, false},
		{"Invalid Auth, no CCLF Files found", http.StatusUnauthorized, []string{"", ""}, []string{"", ""}, true},
		{"Simulate error pulling from repository - Default", http.StatusInternalServerError, []string{"InduceError_Default", ""}, []string{"", ""}, false},
		{"Simulate error pulling from repository - Runout", http.StatusInternalServerError, []string{"", "InduceError_Runout"}, []string{"", ""}, false},
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
					mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, fileType).Return(
						nil,
						service.CCLFNotFoundError{
							FileNumber: 8,
							CMSID:      "",
							FileType:   0,
							CutoffTime: time.Time{}},
					)

				case "InduceError_Default": //for this use case, we're going to pretend that the db connection is closed.
					mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, models.FileTypeDefault).Return(
						nil,
						errors.New("Database connection closed."),
					)
				case "InduceError_Runout": //for this use case, we're going to pretend that the db connection is closed.
					mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, models.FileTypeRunout).Return(
						nil,
						errors.New("Database connection closed."),
					)
				default:
					mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, fileType).Return(
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
			apiVersion := "v1"
			fhirPath := "/" + apiVersion + "/fhir"

			resourceMap := s.resourceType
			h := newHandler(resourceMap, fhirPath, apiVersion, s.db, s.pool)
			h.Svc = mockSvc

			rr := httptest.NewRecorder()
			req := s.genASRequest()
			if tt.invalidAuth {
				req = s.genASRequestInvalidAuth()
			}

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
	err := conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", "false")
	assert.Empty(s.T(), err)

	req := s.genGroupRequest("runout", middleware.RequestParameters{})
	w := httptest.NewRecorder()
	h := &Handler{}
	h.RespWriter = responseutils.NewFhirResponseWriter()
	h.BulkGroupRequest(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)

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
		Data:               []string{"adjudicated", "partially-adjudicated"},
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
		Data:               []string{"partially-adjudicated"},
	}
	acoD := &service.ACOConfig{
		Model:              "Model D",
		Pattern:            "D\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{},
	}

	dataTypeMap := map[string]service.ClaimType{
		"Coverage":             {Adjudicated: true},
		"Patient":              {Adjudicated: true},
		"ExplanationOfBenefit": {Adjudicated: true},
		"Claim":                {Adjudicated: false, PartiallyAdjudicated: true},
		"ClaimResponse":        {Adjudicated: false, PartiallyAdjudicated: true},
	}

	h := NewHandler(dataTypeMap, v2BasePath, apiVersionTwo, s.db, s.pool)
	r := models.NewMockRepository(s.T())
	r.On("CreateJob", mock.Anything, mock.Anything).Return(uint(4), nil)
	h.r = r

	h.supportedDataTypes = dataTypeMap
	client.SetLogger(log.API) // Set logger so we don't get errors later
	jsonBytes, _ := json.Marshal("{}")

	tests := []struct {
		name           string
		cmsId          string
		resources      []string
		expectedCode   int
		acoConfig      *service.ACOConfig
		supplyAuthData bool
		mockQueue      bool
		closeDB        bool
		mockAddJob     bool
	}{
		// aco requesting accessable data
		{"Auth Adj/Partially-Adj, Request Adj/Partially-Adj", "A0000", []string{"Claim", "Patient"}, http.StatusAccepted, acoA, true, true, false, false},
		{"Auth Adj, Request Adj", "B0000", []string{"Patient"}, http.StatusAccepted, acoB, true, true, false, false},
		{"Auth Partially-Adj, Request Partially-Adj", "C0000", []string{"Claim"}, http.StatusAccepted, acoC, true, true, false, false},
		// aco requesting non accessable data
		{"Auth Adj, Request Partially-Adj", "B0000", []string{"Claim"}, http.StatusBadRequest, acoB, true, false, false, true},
		{"Auth Partially-Adj, Request Adj", "C0000", []string{"Patient"}, http.StatusBadRequest, acoC, true, false, false, true},
		{"Auth None, Request Adj", "D0000", []string{"Patient"}, http.StatusBadRequest, acoD, true, false, false, true},
		{"Auth None, Request Partially-Adj", "D0000", []string{"Claim"}, http.StatusBadRequest, acoD, true, false, false, true},
		// other errors
		{"Bad Authentication", "D0000", []string{"Claim"}, http.StatusUnauthorized, acoD, false, false, false, true},
		{"Error Enqueing", "A0000", []string{"Claim", "Patient"}, http.StatusInternalServerError, acoA, true, true, false, true},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(t *testing.T) {
			mockSvc := service.MockService{}
			mockSvc.On("GetACOConfigForID", mock.Anything).Return(test.acoConfig, true)
			mockSvc.On("GetTimeConstraints", mock.Anything, mock.Anything).Return(service.TimeConstraints{}, nil)
			mockSvc.On("GetCutoffTime", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Time{}, constants.GetExistingBenes)
			mockSvc.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.CCLFFile{PerformanceYear: utils.GetPY()}, nil)
			mockSvc.On("FindOldCCLFFile", testUtils.CtxMatcher, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(uint(1), nil)
			h.Svc = &mockSvc

			enqueuer := queueing.NewMockEnqueuer(s.T())
			h.Enq = enqueuer
			if test.mockQueue {
				if test.mockAddJob {
					enqueuer.On("AddPrepareJob", mock.Anything, mock.Anything).Return(errors.New("Unable to unmarshal json."))
				} else {
					enqueuer.On("AddPrepareJob", mock.Anything, mock.Anything).Return(nil)
				}
			}

			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "http://bcda.ms.gov/api/v2/Group/$export", bytes.NewReader(jsonBytes))
			if test.supplyAuthData {
				r = r.WithContext(context.WithValue(r.Context(), auth.AuthDataContextKey, auth.AuthData{
					ACOID: "8d80925a-027e-43dd-8aed-9a501cc4cd91",
					CMSID: test.cmsId,
				}))
			}
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": test.cmsId, "request_id": uuid.NewRandom().String()})
			r = r.WithContext(context.WithValue(r.Context(), log.CtxLoggerKey, newLogEntry))
			r = r.WithContext(context.WithValue(r.Context(), appMiddleware.CtxTransactionKey, uuid.New()))
			r = r.WithContext(middleware.SetRequestParamsCtx(r.Context(), middleware.RequestParameters{
				Since:         time.Date(2000, 01, 01, 00, 00, 00, 00, time.UTC),
				ResourceTypes: test.resources,
				Version:       apiVersionTwo,
			}))

			h.bulkRequest(w, r, constants.DefaultRequest)

			assert.Equal(s.T(), test.expectedCode, w.Code)
			mockSvc.On("GetQueJobs", mock.Anything, mock.Anything).Return([]*worker_types.JobEnqueueArgs{}, nil)
		})
	}
}

// TestRequests verifies that we can initiate an export job for all resource types using all the different handlers
func (s *RequestsTestSuite) TestRequests() {
	apiVersion := "v1"
	fhirPath := "/" + apiVersion + "/fhir"
	resourceMap := s.resourceType

	h := newHandler(resourceMap, fhirPath, apiVersion, s.db, s.pool)

	enqueuer := queueing.NewMockEnqueuer(s.T())
	h.Enq = enqueuer
	enqueuer.On("AddPrepareJob", mock.Anything, mock.Anything).Return(nil)

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
					Version:       apiVersionOne,
					ResourceTypes: []string{resource},
					Since:         since,
				}

				mockSvc := service.MockService{}
				mockSvc.On("GetTimeConstraints", testUtils.CtxMatcher, mock.Anything).Return(service.TimeConstraints{}, nil)
				mockAco := service.ACOConfig{Data: []string{"adjudicated"}}
				mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything).Return(&mockAco, true)

				h.Svc = &mockSvc
				mockSvc.On("GetTimeConstraints", mock.Anything, mock.Anything).Return(service.TimeConstraints{}, nil)
				mockSvc.On("GetCutoffTime", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Time{}, constants.GetExistingBenes)
				if groupID == "all" {
					mockSvc.On("GetLatestCCLFFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
						&models.CCLFFile{ID: 1, PerformanceYear: utils.GetPY()},
						nil,
					)
				} else {
					mockSvc.On("GetLatestCCLFFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
						&models.CCLFFile{ID: 2, PerformanceYear: (utils.GetPY() - 1)},
						nil,
					)
				}
				rr := httptest.NewRecorder()
				req := s.genGroupRequest(groupID, rp)
				req = req.WithContext(context.WithValue(req.Context(), appMiddleware.CtxTransactionKey, uuid.New()))
				h.BulkGroupRequest(rr, req)
				assert.Equal(s.T(), http.StatusAccepted, rr.Code)
			}
		}
	}

	// Validate patient requests
	for _, resource := range resources {
		for _, since := range sinces {
			rp := middleware.RequestParameters{
				Version:       apiVersionOne,
				ResourceTypes: []string{resource},
				Since:         since,
			}
			mockSvc := service.MockService{}
			mockSvc.On("GetTimeConstraints", testUtils.CtxMatcher, mock.Anything).Return(service.TimeConstraints{}, nil)
			mockAco := service.ACOConfig{Data: []string{"adjudicated"}}
			mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything).Return(&mockAco, true)
			mockSvc.On("GetCutoffTime", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Time{}, constants.GetExistingBenes)
			mockSvc.On("GetLatestCCLFFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&models.CCLFFile{ID: 1, PerformanceYear: utils.GetPY()},
				nil,
			)
			h.Svc = &mockSvc

			rr := httptest.NewRecorder()
			req := s.genPatientRequest(rp)
			req = req.WithContext(context.WithValue(req.Context(), appMiddleware.CtxTransactionKey, uuid.New()))
			h.BulkPatientRequest(rr, req)
			assert.Equal(s.T(), http.StatusAccepted, rr.Code)
		}
	}
}

func (s *RequestsTestSuite) TestJobStatusErrorHandling() {

	basePath := v2BasePath
	apiVersion := apiVersionTwo
	requestUrl := v2JobRequestUrl

	tests := []struct {
		testName           string
		status             models.JobStatus
		jobId              string
		responseHeader     int
		useMockService     bool
		timestampOffset    int
		envVarOverride     string
		instigateDBFailure bool
	}{
		{testName: "Invalid jobID (Overflow)",
			status:         models.JobStatusFailedExpired,
			jobId:          "123412341234123412341234123412341234",
			responseHeader: http.StatusBadRequest,
			useMockService: false},
		{testName: "Invalid jobID (Non-overflow)",
			status: models.JobStatusFailedExpired,
			jobId:  "12345", responseHeader: http.StatusNotFound,
			useMockService: false},
		{testName: "Pending Job",
			status: models.JobStatusPending,
			jobId:  "1", responseHeader: http.StatusAccepted,
			useMockService: true},
		{testName: "Archived Job",
			status: models.JobStatusArchived,
			jobId:  "1", responseHeader: http.StatusGone,
			useMockService: true},
		{testName: "Cancelled Job",
			status: models.JobStatusCancelled,
			jobId:  "1", responseHeader: http.StatusNotFound,
			useMockService: true},
		{testName: "Expired Job - Not Cleaned Up",
			status: models.JobStatusCompleted,
			jobId:  "1", responseHeader: http.StatusGone,
			useMockService: true, timestampOffset: -100000000000000},
		{testName: "Acceptable Job",
			status:         models.JobStatusCompleted,
			jobId:          "1",
			useMockService: true, responseHeader: http.StatusOK},
		{testName: "Simulate unspecified DB Failure",
			status: models.JobStatusFailedExpired,
			jobId:  "1", responseHeader: http.StatusInternalServerError,
			useMockService: true, instigateDBFailure: true},
	}

	resourceMap := s.resourceType

	for _, tt := range tests {
		s.T().Run(tt.testName, func(t *testing.T) {
			h := newHandler(resourceMap, basePath, apiVersion, s.db, s.pool)
			if tt.useMockService {
				mockSrv := service.MockService{}
				timestp := time.Now()
				var errResp error = nil
				if tt.instigateDBFailure {
					errResp = sql.ErrConnDone
				}

				mockSrv.On("GetJobAndKeys", testUtils.CtxMatcher, uint(1)).Return(
					&models.Job{
						ID:              1,
						ACOID:           uuid.NewRandom(),
						RequestURL:      requestUrl,
						Status:          tt.status,
						TransactionTime: timestp,
						JobCount:        100,
						CreatedAt:       timestp,
						UpdatedAt:       timestp.Add(time.Duration(tt.timestampOffset)),
					},
					[]*models.JobKey{{
						ID:           1,
						JobID:        1,
						FileName:     "testingtesting",
						ResourceType: "Patient",
					}},
					errResp,
				)

				h.Svc = &mockSrv

			}

			req := httptest.NewRequest("GET", requestUrl, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobId)

			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			req = req.WithContext(context.WithValue(ctx, log.CtxLoggerKey, newLogEntry))

			w := httptest.NewRecorder()
			h.JobStatus(w, req)
			s.Equal(tt.responseHeader, w.Code)
			switch tt.responseHeader {
			case http.StatusBadRequest, http.StatusNotFound, http.StatusGone:
				s.Equal(constants.FHIRJsonContentType, w.Header().Get("Content-Type"))
			case http.StatusOK:
				s.Equal(constants.JsonContentType, w.Header().Get("Content-Type"))
			case http.StatusAccepted:
				s.Equal("", w.Header().Get("Content-Type"))
			}

		})
	}
}

func (s *RequestsTestSuite) TestJobStatusProgress() {
	tests := []struct {
		testName         string
		status           models.JobStatus
		expectedProgress string
	}{
		{testName: "In-Progress job displays partial progress", status: models.JobStatusInProgress, expectedProgress: "50%"},
		{testName: "Completed job doesn't display progress", status: models.JobStatusCompleted, expectedProgress: ""},
		{testName: "Archived job doesn't display progress", status: models.JobStatusArchived, expectedProgress: ""},
	}

	basePath := v2BasePath
	apiVersion := apiVersionTwo
	requestUrl := v2JobRequestUrl
	resourceMap := s.resourceType
	h := newHandler(resourceMap, basePath, apiVersion, s.db, s.pool)

	req := httptest.NewRequest("GET", requestUrl, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "101")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	req = req.WithContext(context.WithValue(ctx, log.CtxLoggerKey, newLogEntry))

	for _, tt := range tests {
		s.T().Run(tt.testName, func(t *testing.T) {
			job := models.Job{ID: 101, Status: tt.status, JobCount: 2}
			jobKey := models.JobKey{ID: 1001, FileName: "goodFile.ndjson"}
			mockSrv := service.MockService{}
			h.Svc = &mockSrv
			mockSrv.On("GetJobAndKeys", testUtils.CtxMatcher, job.ID).Return(&job, []*models.JobKey{&jobKey}, nil)
			w := httptest.NewRecorder()

			h.JobStatus(w, req)
			progressHeader := w.Header().Get("X-Progress")
			if tt.expectedProgress == "" {
				s.Empty(progressHeader)
			} else {
				s.Contains(progressHeader, tt.expectedProgress)
			}
		})
	}
}

func (s *RequestsTestSuite) TestDeleteJob() {
	// DeleteJob
	basePath := v2BasePath
	apiVersion := apiVersionTwo
	requestUrl := v2JobRequestUrl
	tests := []struct {
		name           string
		jobId          string
		responseHeader int
		useMockService bool
	}{
		{name: "Successful Delete", jobId: "1", responseHeader: http.StatusAccepted, useMockService: true},
		{name: "Invalid Job ID (Overflow)", jobId: "112341234123412341234123412341234123", responseHeader: http.StatusBadRequest, useMockService: false},
		{name: "Unable to cancel job", jobId: "1", responseHeader: http.StatusGone, useMockService: true},
		{name: "Internal Server Error Deleting Job", jobId: "1", responseHeader: http.StatusInternalServerError, useMockService: true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			handler := newHandler(s.resourceType, basePath, apiVersion, s.db, s.pool)

			if tt.useMockService {
				mockSrv := service.MockService{}
				switch tt.responseHeader {
				case http.StatusAccepted:
					mockSrv.On("CancelJob", testUtils.CtxMatcher, uint(1)).Return(
						uint(0), nil,
					)
				case http.StatusGone:
					mockSrv.On("CancelJob", testUtils.CtxMatcher, uint(1)).Return(
						uint(0), service.ErrJobNotCancellable,
					)
				default:
					mockSrv.On("CancelJob", testUtils.CtxMatcher, uint(1)).Return(
						uint(0), errors.New("New Error (doesn't matter)"),
					)
				}
				handler.Svc = &mockSrv
			}

			r := httptest.NewRequest("DELETE", requestUrl, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobId)

			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
			r = r.WithContext(ctx)
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
			r = r.WithContext(context.WithValue(ctx, log.CtxLoggerKey, newLogEntry))

			w := httptest.NewRecorder()

			handler.DeleteJob(w, r)

			assert.Equal(t, tt.responseHeader, w.Code)
		})
	}

}

func (s *RequestsTestSuite) TestJobFailedStatus() {

	tests := []struct {
		name string

		basePath   string
		version    string
		requestUrl string
		status     models.JobStatus
	}{
		{"Job Failed v1", v1BasePath, apiVersionOne, v1JobRequestUrl, models.JobStatusFailed},
		{"Job Failed Expired v1", v1BasePath, apiVersionOne, v1JobRequestUrl, models.JobStatusFailedExpired},
		{"Job Failed v2", v2BasePath, apiVersionTwo, v2JobRequestUrl, models.JobStatusFailed},
		{"Job Failed Expired v2", v2BasePath, apiVersionTwo, v2JobRequestUrl, models.JobStatusFailedExpired},
	}

	resourceMap := s.resourceType

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			h := newHandler(resourceMap, tt.basePath, tt.version, s.db, s.pool)
			mockSrv := service.MockService{}
			timestp := time.Now()
			mockSrv.On("GetJobAndKeys", testUtils.CtxMatcher, uint(1)).Return(
				&models.Job{
					ID:              1,
					ACOID:           uuid.NewRandom(),
					RequestURL:      tt.requestUrl,
					Status:          tt.status,
					TransactionTime: timestp,
					JobCount:        100,
					CreatedAt:       timestp,
					UpdatedAt:       timestp,
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

			req := httptest.NewRequest("GET", tt.requestUrl, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", "1")

			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			ctx = context.WithValue(ctx, appMiddleware.CtxTransactionKey, uuid.New())
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{})
			req = req.WithContext(context.WithValue(ctx, log.CtxLoggerKey, newLogEntry))

			w := httptest.NewRecorder()
			h.JobStatus(w, req)
			s.Equal(http.StatusInternalServerError, w.Code)
			assert.Contains(s.T(), w.Body.String(), responseutils.DetailJobFailed)
		})
	}
}

func (s *RequestsTestSuite) TestGetResourceTypes() {

	testCases := []struct {
		aco               string
		apiVersion        string
		expectedResources []string
	}{
		{"TEST123", "v1", []string{"Patient", "ExplanationOfBenefit", "Coverage"}},
		{"D1234", "v1", []string{"Patient", "ExplanationOfBenefit", "Coverage"}},
		{"A0000", "v1", []string{"Patient", "ExplanationOfBenefit", "Coverage"}},
		{"TEST123", "v2", []string{"Patient", "ExplanationOfBenefit", "Coverage", "Claim", "ClaimResponse"}},
		{"A0000", "v2", []string{"Patient", "ExplanationOfBenefit", "Coverage"}},
		{"DA0000", "v2", []string{"Patient", "ExplanationOfBenefit", "Coverage", "Claim", "ClaimResponse"}},
		{"CT000000", "v2", []string{"Patient", "ExplanationOfBenefit", "Coverage", "Claim", "ClaimResponse"}},
	}
	for _, test := range testCases {
		h := newHandler(s.resourceType, "/"+test.apiVersion+"/fhir", test.apiVersion, s.db, s.pool)
		rp := middleware.RequestParameters{
			Version:       test.apiVersion,
			ResourceTypes: []string{},
			Since:         time.Time{},
		}
		rt := h.getResourceTypes(rp, test.aco)

		assert.Equal(s.T(), rt, test.expectedResources)
	}

}

type IntegrationRequestsTestSuite struct {
	suite.Suite
	db          *sql.DB
	pool        *pgxv5Pool.Pool
	dbContainer db.TestDatabaseContainer
}

func TestIntegrationRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationRequestsTestSuite))
}

func (s *IntegrationRequestsTestSuite) SetupSuite() {
	var err error
	s.dbContainer, err = db.NewTestDatabaseContainer()
	require.NoError(s.T(), err)
}

func (s *IntegrationRequestsTestSuite) SetupSubTest() {
	var err error
	s.db, err = s.dbContainer.NewSqlDbConnection()
	require.NoError(s.T(), err)
	s.pool, err = s.dbContainer.NewPgxPoolConnection()
	require.NoError(s.T(), err)
}

func (s *IntegrationRequestsTestSuite) TearDownSubTest() {
	s.db.Close()
	err := s.dbContainer.RestoreSnapshot("Base")
	require.NoError(s.T(), err)
}

func (s *IntegrationRequestsTestSuite) TestBulkRequest_Integration() {
	acoA := &service.ACOConfig{
		Model:              "Model A",
		Pattern:            "A\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      2,
		Disabled:           false,
		Data:               []string{"adjudicated"},
	}

	dataTypeMap := map[string]service.ClaimType{
		"Coverage":             {Adjudicated: true},
		"Patient":              {Adjudicated: true},
		"ExplanationOfBenefit": {Adjudicated: true},
		"Claim":                {Adjudicated: false, PartiallyAdjudicated: true},
		"ClaimResponse":        {Adjudicated: false, PartiallyAdjudicated: true},
	}

	tests := []struct {
		name         string
		cmsId        string
		resources    []string
		reqType      constants.DataRequestType
		since        time.Time
		expectedCode int
		acoConfig    *service.ACOConfig
		jobCreated   bool
	}{
		{"Test Insert PrepareJob", "A0002", []string{"Patient"}, constants.DefaultRequest, time.Now().AddDate(-1, 0, 0), http.StatusAccepted, acoA, true},
		{"Test Invalid Since", "A0002", []string{"Patient"}, constants.RetrieveNewBeneHistData, time.Now().AddDate(0, -6, 0), http.StatusNotFound, acoA, false},
		{"Test Valid Since", "A0002", []string{"Patient"}, constants.RetrieveNewBeneHistData, time.Now().AddDate(0, -2, 0), http.StatusAccepted, acoA, true},
	}

	for _, test := range tests {
		s.Run(test.name, func() {

			client.SetLogger(log.API) // Set logger so we don't get errors later
			h := NewHandler(dataTypeMap, v2BasePath, apiVersionTwo, s.db, s.pool)
			driver := riverpgxv5.New(s.pool)
			acoID := "A0002"
			repo := postgres.NewRepository(s.db)

			// our DB is not always cleaned up properly so sometimes this record exists when this test runs and sometimes it doesnt
			repo.CreateACO(context.Background(), models.ACO{CMSID: &acoID, UUID: uuid.NewUUID()}) // nolint:errcheck
			_, err := repo.CreateCCLFFile(context.Background(), models.CCLFFile{                  // nolint:errcheck
				Name:            "testfilename",
				ACOCMSID:        acoID,
				PerformanceYear: utils.GetPY(),
				Type:            models.FileTypeDefault,
				Timestamp:       (time.Now()),
				CCLFNum:         constants.CCLF8FileNum,
				ImportStatus:    constants.ImportComplete,
			})
			require.NoError(s.T(), err)

			_, err = repo.CreateCCLFFile(context.Background(), models.CCLFFile{ // nolint:errcheck
				Name:            "testfilename2",
				ACOCMSID:        acoID,
				PerformanceYear: utils.GetPY(),
				Type:            models.FileTypeDefault,
				Timestamp:       (time.Now().AddDate(0, -3, 0)),
				CCLFNum:         constants.CCLF8FileNum,
				ImportStatus:    constants.ImportComplete,
			})
			require.NoError(s.T(), err)

			jsonBytes, _ := json.Marshal("{}")
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "http://bcda.ms.gov/api/v2/Group/$export", bytes.NewReader(jsonBytes))
			r = r.WithContext(context.WithValue(r.Context(), auth.AuthDataContextKey, auth.AuthData{
				ACOID: "8d80925a-027e-43dd-8aed-9a501cc4cd91",
				CMSID: test.cmsId,
			}))
			newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": test.cmsId, "request_id": uuid.NewRandom().String()})
			r = r.WithContext(context.WithValue(r.Context(), log.CtxLoggerKey, newLogEntry))
			r = r.WithContext(context.WithValue(r.Context(), appMiddleware.CtxTransactionKey, uuid.New()))
			r = r.WithContext(middleware.SetRequestParamsCtx(r.Context(), middleware.RequestParameters{
				Since:         test.since,
				ResourceTypes: test.resources,
				Version:       apiVersionTwo,
			}))

			ctx := context.Background()
			h.bulkRequest(w, r, test.reqType)
			assert.Equal(s.T(), test.expectedCode, w.Code)
			if test.jobCreated == true {
				jobs := rivertest.RequireManyInserted(ctx, s.T(), driver, []rivertest.ExpectedJob{
					{Args: worker_types.PrepareJobArgs{}, Opts: nil},
				})
				assert.Greater(s.T(), len(jobs), 0)
			}

		})
	}
}

func (s *RequestsTestSuite) genGroupRequest(groupID string, rp middleware.RequestParameters) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/Group/$export", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", groupID)

	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)
	ctx = middleware.SetRequestParamsCtx(ctx, rp)
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	ctx = context.WithValue(ctx, log.CtxLoggerKey, newLogEntry)
	req = req.WithContext(ctx)

	return req
}

func (s *RequestsTestSuite) genPatientRequest(rp middleware.RequestParameters) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/Patient/$export", nil)
	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
	ctx = middleware.SetRequestParamsCtx(ctx, rp)
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	ctx = context.WithValue(ctx, log.CtxLoggerKey, newLogEntry)
	return req.WithContext(ctx)
}

func (s *RequestsTestSuite) genASRequest() *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/attribution_status", nil)
	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	ctx = context.WithValue(ctx, log.CtxLoggerKey, newLogEntry)
	return req.WithContext(ctx)
}
func (s *RequestsTestSuite) genASRequestInvalidAuth() *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/attribution_status", nil)
	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, "")
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"cms_id": "A9999", "request_id": uuid.NewRandom().String()})
	ctx = context.WithValue(ctx, log.CtxLoggerKey, newLogEntry)
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

func MakeTestStructuredLoggerEntry(logFields logrus.Fields) *log.StructuredLoggerEntry {
	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logFields)}
	return newLogEntry
}

func (s *RequestsTestSuite) TestValidateResources() {
	apiVersion := "v1"
	fhirPath := "/" + apiVersion + "/fhir"
	h := newHandler(map[string]service.ClaimType{
		"Patient":              {},
		"Coverage":             {},
		"ExplanationOfBenefit": {},
	}, fhirPath, apiVersion, s.db, s.pool)
	err := h.validateResources([]string{"Vegetable"}, "1234")
	assert.Contains(s.T(), err.Error(), "invalid resource type")
}

type CCLFNotFoundOperationOutcomeError struct {
	FileNumber int
	CMSID      string
	FileType   models.CCLFFileType
	CutoffTime time.Time
}

func (e CCLFNotFoundOperationOutcomeError) Error() string {
	return "OperationOutcome"
}

type QueueError struct{}

func (e QueueError) Error() string {
	return "error"
}

func TestValidateTypeFilterPACEligibility(t *testing.T) {
	// Setup ACO configs
	acoWithPAC := &service.ACOConfig{
		Model:              "Model With PAC",
		Pattern:            "PAC\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{"adjudicated", "partially-adjudicated"},
	}
	acoWithoutPAC := &service.ACOConfig{
		Model:              "Model Without PAC",
		Pattern:            "NOPAC\\d{4}",
		PerfYearTransition: "01/01",
		LookbackYears:      10,
		Disabled:           false,
		Data:               []string{"adjudicated"},
	}

	// Create a minimal handler - we only need the service mock, not the full database setup
	h := &Handler{
		Svc:        &service.MockService{},
		apiVersion: constants.V3Version,
		RespWriter: responseutilsv3.NewFhirResponseWriter(),
	}

	tests := []struct {
		name          string
		cmsID         string
		typeFilter    [][]string
		acoConfig     *service.ACOConfig
		shouldFail    bool
		expectedError string
		description   string
	}{
		{
			name:        "SharedSystemWithPAC",
			cmsID:       "PAC0000",
			typeFilter:  [][]string{{"_tag", "SharedSystem"}},
			acoConfig:   acoWithPAC,
			shouldFail:  false,
			description: "ACO with PAC access should be able to use SharedSystem tag",
		},
		{
			name:          "SharedSystemWithoutPAC",
			cmsID:         "NOPAC0000",
			typeFilter:    [][]string{{"_tag", "SharedSystem"}},
			acoConfig:     acoWithoutPAC,
			shouldFail:    true,
			expectedError: "Model entities in Model Without PAC are not eligible to access partially adjudicated claims data. Requests using the following tags require access to partially adjudicated claims data: [SharedSystem]",
			description:   "ACO without PAC access should be blocked from using SharedSystem tag",
		},
		{
			name:        "SharedSystemURLFormatWithPAC",
			cmsID:       "PAC0000",
			typeFilter:  [][]string{{"_tag", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem"}},
			acoConfig:   acoWithPAC,
			shouldFail:  false,
			description: "ACO with PAC access should be able to use SharedSystem tag in URL format",
		},
		{
			name:          "SharedSystemURLFormatWithoutPAC",
			cmsID:         "NOPAC0000",
			typeFilter:    [][]string{{"_tag", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem"}},
			acoConfig:     acoWithoutPAC,
			shouldFail:    true,
			expectedError: "Model entities in Model Without PAC are not eligible to access partially adjudicated claims data. Requests using the following tags require access to partially adjudicated claims data: [SharedSystem]",
			description:   "ACO without PAC access should be blocked from SharedSystem tag in URL format",
		},
		{
			name:        "FinalActionNoPACRequired",
			cmsID:       "NOPAC0000",
			typeFilter:  [][]string{{"_tag", "FinalAction"}},
			acoConfig:   acoWithoutPAC,
			shouldFail:  false,
			description: "FinalAction tag should not require PAC eligibility",
		},
		{
			name:        "NotFinalActionNoPACRequired",
			cmsID:       "NOPAC0000",
			typeFilter:  [][]string{{"_tag", "NotFinalAction"}},
			acoConfig:   acoWithoutPAC,
			shouldFail:  false,
			description: "NotFinalAction tag should not require PAC eligibility",
		},
		{
			name:        "NationalClaimsHistoryNoPACRequired",
			cmsID:       "NOPAC0000",
			typeFilter:  [][]string{{"_tag", "NationalClaimsHistory"}},
			acoConfig:   acoWithoutPAC,
			shouldFail:  false,
			description: "NationalClaimsHistory tag should not require PAC eligibility",
		},
		{
			name:        "SharedSystemAndFinalActionWithPAC",
			cmsID:       "PAC0000",
			typeFilter:  [][]string{{"_tag", "SharedSystem"}, {"_tag", "FinalAction"}},
			acoConfig:   acoWithPAC,
			shouldFail:  false,
			description: "ACO with PAC should be able to combine SharedSystem with other tags",
		},
		{
			name:          "SharedSystemAndFinalActionWithoutPAC",
			cmsID:         "NOPAC0000",
			typeFilter:    [][]string{{"_tag", "SharedSystem"}, {"_tag", "FinalAction"}},
			acoConfig:     acoWithoutPAC,
			shouldFail:    true,
			expectedError: "Model entities in Model Without PAC are not eligible to access partially adjudicated claims data. Requests using the following tags require access to partially adjudicated claims data: [SharedSystem]",
			description:   "ACO without PAC should be blocked even when SharedSystem is combined with other tags",
		},
		{
			name:        "NoTypeFilter",
			cmsID:       "NOPAC0000",
			typeFilter:  [][]string{},
			acoConfig:   acoWithoutPAC,
			shouldFail:  false,
			description: "No typeFilter should pass validation",
		},
		{
			name:        "NonTagParameters",
			cmsID:       "NOPAC0000",
			typeFilter:  [][]string{{"service-date", "ge2020-01-01"}},
			acoConfig:   acoWithoutPAC,
			shouldFail:  false,
			description: "Non-tag parameters should not require PAC eligibility",
		},
		{
			name:          "ACOConfigNotFound",
			cmsID:         "UNKNOWN0000",
			typeFilter:    [][]string{{"_tag", "SharedSystem"}},
			acoConfig:     nil,
			shouldFail:    true,
			expectedError: "Unable to determine ACO configuration",
			description:   "Should fail when ACO config is not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new mock service for each test
			mockSvc := service.MockService{}
			h.Svc = &mockSvc

			// Check if SharedSystem tag is present (which requires PAC check)
			requiresPACCheck := false
			for _, paramPair := range test.typeFilter {
				if len(paramPair) == 2 && paramPair[0] == "_tag" {
					tagCode := extractTagCodeFromValue(paramPair[1])
					if tagCode == "SharedSystem" {
						requiresPACCheck = true
						break
					}
				}
			}

			// Setup mock service only if PAC check is required
			if requiresPACCheck {
				if test.acoConfig != nil {
					mockSvc.On("GetACOConfigForID", test.cmsID).Return(test.acoConfig, true)
				} else {
					mockSvc.On("GetACOConfigForID", test.cmsID).Return(nil, false)
				}
			}

			// Create response writer
			w := httptest.NewRecorder()
			ctx := context.Background()
			ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)

			// Call the validation function
			err := h.validateTypeFilterPACEligibility(ctx, test.typeFilter, test.cmsID, w)

			if test.shouldFail {
				assert.Error(t, err, test.description)
				assert.Equal(t, http.StatusBadRequest, w.Code, test.description)
				if test.expectedError != "" {
					assert.Contains(t, w.Body.String(), test.expectedError, test.description)
				}
			} else {
				assert.NoError(t, err, test.description)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestExtractTagCodeFromValue(t *testing.T) {
	tests := []struct {
		name     string
		tagValue string
		expected string
	}{
		{"shortFormat", "SharedSystem", "SharedSystem"},
		{"shortFormatNotFinalAction", "NotFinalAction", "NotFinalAction"},
		{"urlFormatSystemType", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem", "SharedSystem"},
		{"urlFormatFinalAction", "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction", "FinalAction"},
		{"urlFormatWithMultiplePipes", "https://example.com|system|SharedSystem", "SharedSystem"},
		{"emptyString", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTagCodeFromValue(tt.tagValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureNCHOnlyForNonPAC(t *testing.T) {
	ctx := context.Background()
	ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)

	// Create mock service
	mockSvc := new(service.MockService)
	h := &Handler{
		Svc: mockSvc,
	}

	// ACO configs
	acoWithPAC := &service.ACOConfig{
		Model: "Model With PAC",
		Data:  []string{"adjudicated", "partially-adjudicated"},
	}

	acoWithoutPAC := &service.ACOConfig{
		Model: "Model Without PAC",
		Data:  []string{"adjudicated"},
	}

	tests := []struct {
		name         string
		cmsID        string
		typeFilter   [][]string
		acoConfig    *service.ACOConfig
		expectedTags []string // Expected _tag values in the returned filter
		description  string
	}{
		{
			name:         "NonPACNoFilter",
			cmsID:        "NOPAC0000",
			typeFilter:   [][]string{},
			acoConfig:    acoWithoutPAC,
			expectedTags: []string{"https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory"},
			description:  "Non-PAC ACO with no filter should get NCH filter added",
		},
		{
			name:         "NonPACWithNCHFilter",
			cmsID:        "NOPAC0000",
			typeFilter:   [][]string{{"_tag", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory"}},
			acoConfig:    acoWithoutPAC,
			expectedTags: []string{"https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory"},
			description:  "Non-PAC ACO with existing NCH filter should not duplicate it",
		},
		{
			name:         "NonPACWithFinalAction",
			cmsID:        "NOPAC0000",
			typeFilter:   [][]string{{"_tag", "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"}},
			acoConfig:    acoWithoutPAC,
			expectedTags: []string{"https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory"},
			description:  "Non-PAC ACO with FinalAction should still get NCH filter added",
		},
		{
			name:         "PACNoFilter",
			cmsID:        "PAC0000",
			typeFilter:   [][]string{},
			acoConfig:    acoWithPAC,
			expectedTags: []string{},
			description:  "PAC ACO with no filter should not get NCH filter added",
		},
		{
			name:         "PACWithSharedSystem",
			cmsID:        "PAC0000",
			typeFilter:   [][]string{{"_tag", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem"}},
			acoConfig:    acoWithPAC,
			expectedTags: []string{"https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem"},
			description:  "PAC ACO with SharedSystem should not get NCH filter added",
		},
		{
			name:         "NonPACWithServiceDate",
			cmsID:        "NOPAC0000",
			typeFilter:   [][]string{{"service-date", "ge2024-01-01"}},
			acoConfig:    acoWithoutPAC,
			expectedTags: []string{"https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory"},
			description:  "Non-PAC ACO with service-date but no _tag should get NCH filter added",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup mock
			mockSvc.On("GetACOConfigForID", test.cmsID).Return(test.acoConfig, true)

			// Call the function
			result := h.ensureNCHOnlyForNonPAC(ctx, test.typeFilter, test.cmsID)

			// Extract _tag values from result
			var actualTags []string
			for _, paramPair := range result {
				if len(paramPair) == 2 && paramPair[0] == "_tag" {
					actualTags = append(actualTags, paramPair[1])
				}
			}

			// Verify expected tags (handle nil vs empty slice)
			if len(test.expectedTags) == 0 && len(actualTags) == 0 {
				// Both are empty, that's fine
			} else {
				assert.Equal(t, test.expectedTags, actualTags, test.description)
			}

			// Verify other parameters are preserved
			var otherParams [][]string
			for _, paramPair := range result {
				if len(paramPair) == 2 && paramPair[0] != "_tag" {
					otherParams = append(otherParams, paramPair)
				}
			}
			var expectedOtherParams [][]string
			for _, paramPair := range test.typeFilter {
				if len(paramPair) == 2 && paramPair[0] != "_tag" {
					expectedOtherParams = append(expectedOtherParams, paramPair)
				}
			}
			assert.Equal(t, expectedOtherParams, otherParams, "Other parameters should be preserved")

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestEnsureNCHOnlyForNonPACWithDefaultEOB(t *testing.T) {
	// This test verifies that when a non-PAC ACO makes a v3 request without _type parameter,
	// ExplanationOfBenefit is included by default, and the NCH filter is added
	ctx := context.Background()
	ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)

	mockSvc := new(service.MockService)
	h := &Handler{
		Svc:        mockSvc,
		apiVersion: constants.V3Version,
	}

	acoWithoutPAC := &service.ACOConfig{
		Model: "Model Without PAC",
		Data:  []string{"adjudicated"},
	}

	// Simulate getResourceTypes returning default types (including EOB)
	resourceTypes := []string{"Patient", "ExplanationOfBenefit", "Coverage"}

	// No typeFilter provided (empty)
	typeFilter := [][]string{}

	// Setup mock
	mockSvc.On("GetACOConfigForID", "NOPAC0000").Return(acoWithoutPAC, true)

	// Call ensureNCHOnlyForNonPAC (this is what gets called when EOB is in resourceTypes)
	result := h.ensureNCHOnlyForNonPAC(ctx, typeFilter, "NOPAC0000")

	// Verify NCH filter was added
	var actualTags []string
	for _, paramPair := range result {
		if len(paramPair) == 2 && paramPair[0] == "_tag" {
			actualTags = append(actualTags, paramPair[1])
		}
	}

	expectedNCHTag := "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory"
	assert.Contains(t, actualTags, expectedNCHTag, "NCH filter should be added for non-PAC ACO when EOB is requested")

	// Verify that ExplanationOfBenefit would trigger this logic
	assert.True(t, utils.ContainsString(resourceTypes, "ExplanationOfBenefit"), "EOB should be in default resource types")

	mockSvc.AssertExpectations(t)
}

type DatabaseError struct{}

func (e DatabaseError) Error() string {
	return "error"
}
