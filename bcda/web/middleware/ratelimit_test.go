package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"
	logAPI "github.com/CMSgov/bcda-app/log"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RateLimitMiddlewareTestSuite struct {
	suite.Suite
	db *sql.DB
}

func TestRateLimitMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(RateLimitMiddlewareTestSuite))
}
func (s *RateLimitMiddlewareTestSuite) SetupSuite() {
	s.db = database.Connect()
}
func (s *RateLimitMiddlewareTestSuite) TestNoConcurrentJobs() {
	cfg := &service.Config{RateLimitConfig: service.RateLimitConfig{All: true}}
	middleware := NewRateLimitMiddleware(cfg, s.db)
	tests := []struct {
		name string
		rp   RequestParameters
		jobs []*models.Job
	}{
		{"NoJobs", RequestParameters{}, nil},
		{"DifferentVersion", RequestParameters{Version: "v2"},
			[]*models.Job{{RequestURL: constants.V1Path + constants.PatientExportPath}}},
		{"DifferentType", RequestParameters{Version: "v1", ResourceTypes: []string{"Coverage"}},
			[]*models.Job{{RequestURL: constants.V1Path + constants.PatientExportPath}}},
		{"JobGroupExportJustPatient", RequestParameters{ResourceTypes: []string{"Patient"}, Version: "v2", RequestURL: "/v2/Group/all/$export?_type=Patient"},
			[]*models.Job{{RequestURL: constants.V2Path + constants.PatientEOBPath, CreatedAt: time.Now()}}},
	}

	for _, tt := range tests {
		mockRepo := &models.MockRepository{}
		mockRepo.On("GetJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			tt.jobs,
			nil,
		)
		middleware.repository = mockRepo

		rr := httptest.NewRecorder()
		middleware.CheckConcurrentJobs(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Conncurrent job test route check, blank return for overrides
		})).ServeHTTP(rr, getRequest(tt.rp))

		assert.Equal(s.T(), http.StatusOK, rr.Code)
	}
}
func (s *RateLimitMiddlewareTestSuite) TestHasConcurrentJobs() {
	cfg := &service.Config{RateLimitConfig: service.RateLimitConfig{All: true}}
	middleware := NewRateLimitMiddleware(cfg, s.db)
	// These jobs are not considered when determine duplicate jobs
	ignoredJobs := []*models.Job{
		{RequestURL: "http://a{b"},                           // InvalidURL
		{RequestURL: "/api/Patient/$export?_noversion=true"}, // No version specified
		{RequestURL: "/api/v1/Patient/$export?_jobtimeout=true", CreatedAt: time.Now().Add(-365 * 24 * time.Hour)}, // Too old
		{RequestURL: "/api/v1/Patient/$export?_type=Patient", CreatedAt: time.Now()},                               // Different resource type
	}

	tests := []struct {
		name           string
		rp             RequestParameters
		additionalJobs []*models.Job
	}{
		{"RequestForAllResources", RequestParameters{ResourceTypes: nil, Version: "v1"}, nil},
		{"DuplicateType", RequestParameters{ResourceTypes: []string{"Patient"}, Version: "v1"}, nil},
		{"JobAllResources", RequestParameters{ResourceTypes: []string{"Patient"}, Version: "v1"},
			[]*models.Job{{RequestURL: constants.V1Path + constants.PatientExportPath, CreatedAt: time.Now()}}},
		{"JobGroupExportDuplicateAll", RequestParameters{ResourceTypes: nil, Version: "v2", RequestURL: "/api/v2/Group/all/$export"},
			[]*models.Job{{RequestURL: constants.V2Path + constants.GroupExportPath, CreatedAt: time.Now()}}},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockRepo := &models.MockRepository{}
			mockRepo.On("GetJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				append(ignoredJobs, tt.additionalJobs...),
				nil,
			)
			middleware.repository = mockRepo

			rr := httptest.NewRecorder()
			middleware.CheckConcurrentJobs(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				// Conncurrent job test route check, blank return for overrides
			})).ServeHTTP(rr, getRequest(tt.rp))

			assert.NotEmpty(s.T(), rr.Header().Get("Retry-After"))
		})
	}
}

func (s *RateLimitMiddlewareTestSuite) TestFailedToGetJobs() {
	cfg := &service.Config{RateLimitConfig: service.RateLimitConfig{All: true}}
	middleware := NewRateLimitMiddleware(cfg, s.db)
	mockRepo := &models.MockRepository{}
	mockRepo.On("GetJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		errors.New("FORCING SOME ERROR"),
		nil,
	)
	middleware.repository = mockRepo

	rr := httptest.NewRecorder()
	middleware.CheckConcurrentJobs(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Conncurrent job test route check, blank return for overrides
	})).ServeHTTP(rr, getRequest(RequestParameters{}))

	assert.Equal(s.T(), http.StatusInternalServerError, rr.Code)
	assert.Contains(s.T(), rr.Body.String(), "code\":\"exception\"")
}

func getRequest(rp RequestParameters) *http.Request {
	ctx := context.WithValue(context.Background(), auth.AuthDataContextKey, auth.AuthData{ACOID: uuid.New()})
	ctx = SetRequestParamsCtx(ctx, rp)
	ctx = logAPI.NewStructuredLoggerEntry(log.New(), ctx)
	// Since we're supplying the request parameters in the context, the actual req URL does not matter
	return httptest.NewRequest("GET", "/api/v1/Patient", nil).WithContext(ctx)
}

func (s *RateLimitMiddlewareTestSuite) TestHasDuplicatesFullString() {
	ctx := context.Background()
	cfg := &service.Config{RateLimitConfig: service.RateLimitConfig{All: true}}
	middleware := NewRateLimitMiddleware(cfg, s.db)
	ctx = logAPI.NewStructuredLoggerEntry(log.New(), ctx)
	otherJobs := []*models.Job{
		{ID: 1, RequestURL: "https://api.abcd.123.net/api/v2/Group/runout/$export?_since=2024-02-11T00%3A00%3A00.0000-00%3A00&_type=Patient%2CCoverage%2CExplanationOfBenefit", CreatedAt: time.Now(), Status: models.JobStatusPending},
		{ID: 2, RequestURL: "http://localhost:3000/api/v2/Group/all/$export?_since=2024-02-15T00%3A00%3A00.0000-00%3A00&_type=Patient%2CCoverage%2CExplanationOfBenefit", CreatedAt: time.Now().Add(-3 * 24 * time.Hour), Status: models.JobStatusExpired},
		{ID: 1, RequestURL: "https://api.abcd.123.net/api/v2/Group/runout/$export?_since=2024-02-11T00%3A00%3A00.0000-00%3A00", CreatedAt: time.Now(), Status: models.JobStatusPending},
	}

	tests := []struct {
		name          string
		rp            RequestParameters
		expectedValue bool
	}{
		{"TestUnparseableNewJob", RequestParameters{ResourceTypes: nil, Version: "v2", RequestURL: "/path%zz%20with%20spaces?query=value"}, false},
		{"TestNewDuplicateJobWithEscaping", RequestParameters{ResourceTypes: []string{"Patient", "Coverage", "ExplanationOfBenefit"}, Version: "v2", RequestURL: "/api/v2/Group/runout/$export?_since=2024-02-11T00%3A00%3A00.0000-00%3A00&_type=Patient%2CCoverage%2CExplanationOfBenefit"}, true},
		{"TestNewDuplicateJobSubsetOfTypes", RequestParameters{ResourceTypes: []string{"Patient"}, Version: "v2", RequestURL: "/api/v2/Group/runout/$export?_since=2024-02-11T04%3A00%3A00.0000-00%3A00&_type=Patient"}, true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			responseBool := middleware.hasDuplicates(ctx, otherJobs, tt.rp.ResourceTypes, tt.rp.Version, tt.rp.RequestURL)
			assert.Equal(s.T(), tt.expectedValue, responseBool)
		})
	}
}

func (s *RateLimitMiddlewareTestSuite) TestShouldRateLimit() {
	cmsID := "MyFavoriteACO"
	otherCMSID := "OtherCMSID"

	tests := []struct {
		name          string
		cmsID         string
		config        service.RateLimitConfig
		expectedValue bool
	}{
		{"Apply rate limit for all requests", cmsID, service.RateLimitConfig{All: true, ACOs: []string{}}, true},
		{"Apply rate limit for no requests", cmsID, service.RateLimitConfig{All: false, ACOs: []string{}}, false},
		{"Apply rate limit for ACO in limit list", cmsID, service.RateLimitConfig{All: false, ACOs: []string{cmsID, otherCMSID}}, true},
		{"Dont apply rate limit for ACO not in limit list", cmsID, service.RateLimitConfig{All: false, ACOs: []string{otherCMSID}}, false},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			actualValue := shouldRateLimit(tt.config, tt.cmsID)
			assert.Equal(s.T(), tt.expectedValue, actualValue, tt.name)
		})
	}
}
