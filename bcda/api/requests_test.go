package api

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
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
	s.db = database.GetDbConnection()

	// Create an ACO for us to test with
	s.acoID = uuid.NewUUID()
	cmsID := "ZYXWV"

	postgrestest.CreateACO(s.T(), s.db, models.ACO{UUID: s.acoID, CMSID: &cmsID})
}

func (s *RequestsTestSuite) SetupTest() {
	s.runoutEnabledEnvVar = os.Getenv("BCDA_ENABLE_RUNOUT")
}

func (s *RequestsTestSuite) TearDownSuite() {
	postgrestest.DeleteJobsByACOID(s.T(), s.db, s.acoID)
	postgrestest.DeleteACO(s.T(), s.db, s.acoID)
	s.db.Close()
}

func (s *RequestsTestSuite) TearDownTest() {
	os.Setenv("BCDA_ENABLE_RUNOUT", s.runoutEnabledEnvVar)
}

func (s *RequestsTestSuite) TestRunoutEnabled() {
	os.Setenv("BCDA_ENABLE_RUNOUT", "true")
	qj := []*que.Job{{Type: "ProcessJob"}, {Type: "ProcessJob"}}
	tests := []struct {
		name string

		errToReturn error
		respCode    int
	}{
		{"Successful", nil, http.StatusAccepted},
		{"No CCLF file found", models.CCLFNotFoundError{}, http.StatusNotFound},
		{"Some other error", errors.New("Some other error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &models.MockService{}
			var jobs []*que.Job
			if tt.errToReturn == nil {
				jobs = qj
			}

			mockSvc.On("GetQueJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(jobs, tt.errToReturn)
			h := NewHandler([]string{"ExplanationOfBenefit", "Coverage", "Patient"}, "/v1/fhir")
			h.Svc = mockSvc

			req := s.genGroupRequest("runout")
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

func (s *RequestsTestSuite) TestRunoutDisabled() {
	os.Setenv("BCDA_ENABLE_RUNOUT", "false")
	req := s.genGroupRequest("runout")
	w := httptest.NewRecorder()
	h := &Handler{}
	h.BulkGroupRequest(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Contains(string(body), "Invalid group ID")
}

func (s *RequestsTestSuite) TestInvalidRequests() {
	supportedTypes := []string{"ExplanationOfBenefit", "Coverage", "Patient"}
	h := NewHandler(supportedTypes, "/v1/fhir")

	type reqParams struct {
		types        []string
		since        string
		outputFormat string
	}
	tests := []struct {
		name             string
		reqParams        reqParams
		extraQueryParams map[string]string
		errMsg           string
	}{
		{"_elements query parameter", reqParams{}, map[string]string{"_elements": "blah,blah,blah"}, "Invalid parameter: this server does not support the _elements parameter."},

		{"Unsupported type", reqParams{types: []string{"Practitioner"}}, nil, "Invalid resource type"},
		{"Duplicate types", reqParams{types: []string{"Patient", "Patient"}}, nil, "Repeated resource type"},

		{"Invalid since", reqParams{since: "01-01-2020"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (non-date)", reqParams{since: "invalidDate"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (in the future)", reqParams{since: "2100-01-01T00:00:00Z"}, nil, "Invalid date format supplied in _since parameter. Date must be a date that has already passed"},
		{"Invalid since (escape character format)", reqParams{since: "2020-03-01T00%3A%2000%3A00.000-00%3A00"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (missing timezone)", reqParams{since: "2020-02-13T08:00:00.000"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (invalid time)", reqParams{since: "2020-02-13T33:00:00.000-05:00"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (invalid date)", reqParams{since: "2020-20-13T08:00:00.000-05:00"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (no time)", reqParams{since: "2020-03-01"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (invalid date no time)", reqParams{since: "2020-04-0"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (junk after time)", reqParams{since: "2020-02-13T08:00:00.000-05:00dfghj"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},
		{"Invalid since (junk before date)", reqParams{since: "erty2020-02-13T08:00:00.000-05:00"}, nil, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."},

		{"Invalid output format (text/html)", reqParams{outputFormat: "text/html"}, nil, "_outputFormat parameter must be application/fhir+ndjson, application/ndjson, or ndjson"},
		{"Invalid output format (application/xml)", reqParams{outputFormat: "application/xml"}, nil, "_outputFormat parameter must be application/fhir+ndjson, application/ndjson, or ndjson"},
		{"Invalid output format (x-custom)", reqParams{outputFormat: "x-custom"}, nil, "_outputFormat parameter must be application/fhir+ndjson, application/ndjson, or ndjson"},
		{"Invalid query parameter (extra ?)", reqParams{}, map[string]string{"?_since": "2020-09-13T08:00:00.000-05:00"}, "Invalid parameter: query parameters cannot start with ?"},
	}

	for _, tt := range tests {
		u, err := url.Parse("/api/v1")
		if err != nil {
			s.Failf("Failed to parse URL %s", err.Error())
		}

		q := u.Query()
		if len(tt.reqParams.types) > 0 {
			q.Set("_type", strings.Join(tt.reqParams.types, ","))
		}
		if len(tt.reqParams.since) > 0 {
			q.Set("_since", tt.reqParams.since)
		}
		if len(tt.reqParams.outputFormat) > 0 {
			q.Set("_outputFormat", tt.reqParams.outputFormat)
		}
		for key, value := range tt.extraQueryParams {
			q.Set(key, value)
		}

		u.RawQuery = q.Encode()
		req := httptest.NewRequest("GET", u.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("groupId", "all")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		s.T().Run(fmt.Sprintf("%s-group", tt.name), func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.BulkGroupRequest(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})

		s.T().Run(fmt.Sprintf("%s-patient", tt.name), func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.BulkGroupRequest(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})
	}
}

func (s *RequestsTestSuite) TestCheck429() {
	validJob := models.Job{RequestURL: "/api/v1/Group/$export", Status: models.JobStatusInProgress, Model: gorm.Model{CreatedAt: time.Now()}}
	expiredJob := models.Job{RequestURL: "/api/v1/Group/$export", Status: models.JobStatusInProgress, Model: gorm.Model{CreatedAt: time.Now().Add(-2 * GetJobTimeout())}}
	duplicateType := models.Job{RequestURL: "/api/v1/Group/$export?_type=Patient", Status: models.JobStatusInProgress, Model: gorm.Model{CreatedAt: time.Now()}}
	tests := []struct {
		name        string
		job         models.Job
		version     string
		passesCheck bool
	}{
		{"Same version", validJob, "v1", false},
		{"Different version", validJob, "v2", true},
		{"Invalid job (bad URL)", models.Job{RequestURL: string([]byte{0x7f})}, "", false},
		{"Invalid job (no version)", models.Job{RequestURL: "/api/Group/$export"}, "", false},
		{"Same version - expired", expiredJob, "v1", true},
		{"Duplicate type", duplicateType, "v1", false},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			res, err := check429([]*models.Job{&tt.job}, []string{"Patient"}, tt.version)
			if tt.passesCheck {
				assert.NotNil(t, res)
				assert.NoError(t, err)
			} else {
				assert.Nil(t, res)
				assert.Error(t, err)
			}
		})
	}
}

// TestBulkRequestWithOldJobPaths verifies that bulk requests for a caller with older jobs
// whose paths are no longer valid will still be accepted.
func (s *RequestsTestSuite) TestBulkRequestWithOldJobPaths() {
	// Switch over to validate the 429 behavior
	dt := "DEPLOYMENT_TARGET"
	target := os.Getenv(dt)
	defer os.Setenv(dt, target)
	os.Setenv(dt, "prod")

	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	postgrestest.CreateJobs(s.T(), s.db,
		&models.Job{ACOID: aco.UUID, RequestURL: "https://api.bcda.cms.gov/api/v1/Coverage/$export", Status: models.JobStatusFailed},
		&models.Job{ACOID: aco.UUID, RequestURL: "https://api.bcda.cms.gov/api/v1/ExplanationOfBenefit/$export", Status: models.JobStatusExpired},
		&models.Job{ACOID: aco.UUID, RequestURL: "https://api.bcda.cms.gov/api/v1/SomeCoolUnsupportedResource/$export", Status: models.JobStatusCompleted},
	)
	// Create jobs that githave an unsupported path

	resources := []string{"ExplanationOfBenefit", "Coverage", "Patient"}
	mockSvc := &models.MockService{}
	mockSvc.On("GetQueJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	h := NewHandler(resources, "/v1/fhir")
	h.Svc = mockSvc

	req := s.genGroupRequest("all")
	w := httptest.NewRecorder()
	h.bulkRequest(resources, w, req, models.RetrieveNewBeneHistData)

	s.Equal(http.StatusAccepted, w.Result().StatusCode)
}

func (s *RequestsTestSuite) genGroupRequest(groupID string) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/Group/$export", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", groupID)

	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	return req
}
