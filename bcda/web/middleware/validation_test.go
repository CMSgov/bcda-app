package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var noop http.HandlerFunc = func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(http.StatusOK) }

func TestValidRequestURL(t *testing.T) {
	// Allow us to retrieve the RequestParameters by grabing the updated context.
	// When we call *http.Request.WithContext(ctx), a new request is created.
	// So we cannot leverage the context associated with the original request
	var ctx context.Context
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})

	now := time.Now().Add(-24 * time.Hour).Round(time.Millisecond)
	req, err := http.NewRequest("GET",
		fmt.Sprintf("/api/v1/Patient/$export?_type=Patient&_since=%s&_outputFormat=ndjson",
			now.Format(time.RFC3339Nano)),
		nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	ValidateRequestURL(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify we have the context as expected
	rp, ok := RequestParametersFromContext(ctx)
	assert.True(t, ok)
	// assert.True(t, now.Equal(rp.Since), "Since parameter does not match")
	assert.Equal(t, rp.ResourceTypes, []string{"Patient"})
	assert.Equal(t, rp.Version, "v1")
}

func TestInvalidRequestURL(t *testing.T) {

	base := "/api/v1/Patient/$export?"
	tests := []struct {
		name   string
		url    string
		errMsg string
	}{
		{"invalidOutputFormat", fmt.Sprintf("%s_outputFormat=invalid", base), "_outputFormat parameter must be one of"},
		{"elementsNotSupported", fmt.Sprintf("%s_elements=invalid", base), "does not support the _elements parameter"},
		{"contains?", fmt.Sprintf("%s?_type=Patient", base), "query parameters cannot start with ?"},
		{"invalidSince", fmt.Sprintf("%s_since=05-25-1977", base), "Date must be in FHIR Instant format"},
		{"futureSince", fmt.Sprintf("%s_since=%s", base, time.Now().Add(24*time.Hour).Format(time.RFC3339Nano)),
			"Date must be a date that has already passed"},
		{"repeatedType", fmt.Sprintf("%s_type=Patient,Patient", base), "Repeated resource type Patient"},
		{"noVersion", "/api/Patient$export", "cannot retrieve version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			assert.NoError(t, err)
			rr := httptest.NewRecorder()
			ValidateRequestURL(noop).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})
	}
}

func TestValidRequestHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
	assert.NoError(t, err)
	req.Header.Set("Accept", "application/fhir+json")
	req.Header.Set("Prefer", "respond-async")

	rr := httptest.NewRecorder()
	ValidateRequestHeaders(noop).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
func TestInvalidRequestHeaders(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		preferHeader string
		errMsg       string
	}{
		{"NoAcceptHeader", "", "respond-async", "Accept header is required"},
		{"InvalidAcceptHeader", "invalid", "respond-async", "application/fhir+json is the only supported response format"},
		{"NoPreferHeader", "application/fhir+json", "", "Prefer header is required"},
		{"InvalidPreferHeader", "application/fhir+json", "invalid", "Only asynchronous responses are supported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
			assert.NoError(t, err)
			req.Header.Set("Accept", tt.acceptHeader)
			req.Header.Set("Prefer", tt.preferHeader)

			rr := httptest.NewRecorder()
			ValidateRequestHeaders(noop).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})
	}
}
