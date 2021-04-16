package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNoConcurrentJobs(t *testing.T) {
	tests := []struct {
		name string
		rp   RequestParameters
		jobs []*models.Job
	}{
		{"NoJobs", RequestParameters{}, nil},
		{"DifferentVersion", RequestParameters{Version: "v2"},
			[]*models.Job{{RequestURL: "/api/v1/Patient/$export"}}},
		{"DifferentType", RequestParameters{Version: "v1", ResourceTypes: []string{"Coverage"}},
			[]*models.Job{{RequestURL: "/api/v1/Patient/$export"}}},
	}

	for _, tt := range tests {
		mockRepo := &models.MockRepository{}
		// ctx, acoID, inprogress, pending
		mockRepo.On("GetJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			tt.jobs, //jobs
			nil,     //error
		)
		repository = mockRepo

		rr := httptest.NewRecorder()
		CheckConcurrentJobs(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})).
			ServeHTTP(rr, getRequest(tt.rp))
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}
func TestHasConcurrentJobs(t *testing.T) {
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
			[]*models.Job{{RequestURL: "/api/v1/Patient/$export", CreatedAt: time.Now()}}},
	}

	for _, tt := range tests {
		mockRepo := &models.MockRepository{}
		// ctx, acoID, inprogress, pending
		mockRepo.On("GetJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			append(ignoredJobs, tt.additionalJobs...), //jobs
			nil, //error
		)
		repository = mockRepo

		rr := httptest.NewRecorder()
		CheckConcurrentJobs(nil).ServeHTTP(rr, getRequest(tt.rp))
		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("Retry-After"))
	}
}

func TestFailedToGetJobs(t *testing.T) {
	mockRepo := &models.MockRepository{}
	// ctx, acoID, inprogress, pending
	mockRepo.On("GetJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		errors.New("FORCING SOME ERROR"),
		nil, //error
	)
	repository = mockRepo

	rr := httptest.NewRecorder()
	CheckConcurrentJobs(nil).ServeHTTP(rr, getRequest(RequestParameters{}))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "code\":\"exception\"")
}

func getRequest(rp RequestParameters) *http.Request {
	ctx := context.WithValue(context.Background(), auth.AuthDataContextKey, auth.AuthData{ACOID: uuid.New()})
	ctx = NewRequestParametersContext(ctx, rp)
	// Since we're supplying the request parameters in the context, the actual req URL does not matter
	return httptest.NewRequest("GET", "/api/v1/Patient", nil).WithContext(ctx)
}
