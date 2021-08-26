package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
)

type RequestsTestSuite struct {
	suite.Suite

	runoutEnabledEnvVar string

	db *sql.DB

	acoID uuid.UUID
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

			resourceMap := map[string]service.DataType{
				"Patient":              {Adjudicated: true},
				"Coverage":             {Adjudicated: true},
				"ExplanationOfBenefit": {Adjudicated: true},
			}

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

func (s *RequestsTestSuite) TestAttributionStatus() {
	tests := []struct {
		name string

		errToReturn error
		respCode    int
		fileNames   []string
		fileTypes   []string
	}{
		{"Successful with both files", nil, http.StatusOK, []string{"cclf_test_file_1", "cclf_test_file_2"}, []string{"default", "runout"}},
		{"Successful with default file", nil, http.StatusOK, []string{"cclf_test_file_1", ""}, []string{"default", ""}},
		{"Successful with runout file", nil, http.StatusOK, []string{"", "cclf_test_file_2"}, []string{"", "runout"}},
		{"No CCLF files found", nil, http.StatusNotFound, []string{"", ""}, []string{"", ""}},
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

			resourceMap := map[string]service.DataType{
				"Patient":              {Adjudicated: true},
				"Coverage":             {Adjudicated: true},
				"ExplanationOfBenefit": {Adjudicated: true},
			}
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
				for _, fileStatus := range resp.CCLFFiles {
					if tt.fileNames[count] != "" {
						assert.Equal(s.T(), tt.fileTypes[count], fileStatus.Type)
						assert.Equal(s.T(), tt.fileNames[count], fileStatus.Name)
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

// TestRequests verifies that we can initiate an export job for all resource types using all the different handlers
func (s *RequestsTestSuite) TestRequests() {

	resourceMap := map[string]service.DataType{
		"Patient":              {Adjudicated: true},
		"Coverage":             {Adjudicated: true},
		"ExplanationOfBenefit": {Adjudicated: true},
	}

	h := newHandler(resourceMap, "/v1/fhir", "v1", s.db)

	// Use a mock to ensure that this test does not generate artifacts in the queue for other tests
	enqueuer := &queueing.MockEnqueuer{}
	enqueuer.On("AddJob", mock.Anything, mock.Anything).Return(nil)
	h.Enq = enqueuer
	mockSvc := service.MockService{}

	mockSvc.On("GetQueJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*models.JobEnqueueArgs{}, nil)
	mockAco := service.ACOConfig{
		Data: []string{"adjudicated"},
	}
	mockSvc.On("GetACOConfigForID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockAco, true)

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
