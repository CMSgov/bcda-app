package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var noop http.HandlerFunc = func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(http.StatusOK) }

func TestValidRequestURL(t *testing.T) {
	// Allow us to retrieve the RequestParameters by grabbing the updated context.
	// When we call *http.Request.WithContext(ctx), a new request is created.
	// So we cannot leverage the context associated with the original request
	var ctx context.Context
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})

	now := time.Now().Add(-24 * time.Hour).Round(time.Millisecond)
	req, err := http.NewRequest("GET",
		fmt.Sprintf("/api/v1/Patient/$export?_type=Patient&_since=%s&_outputFormat=ndjson&_typeFilter=ExplanationOfBenefit%%3Fservice-date%%3Dgt2001-04-01",
			now.Format(time.RFC3339Nano)),
		nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	ValidateRequestURL(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify we have the context as expected
	rp, ok := GetRequestParamsFromCtx(ctx)
	assert.True(t, ok)
	// assert.True(t, now.Equal(rp.Since), "Since parameter does not match")
	assert.Equal(t, rp.ResourceTypes, []string{"Patient"})
	assert.Equal(t, rp.Version, "v1")
}

func TestValidv3RequestURL(t *testing.T) {
	// Allow us to retrieve the RequestParameters by grabbing the updated context.
	// When we call *http.Request.WithContext(ctx), a new request is created.
	// So we cannot leverage the context associated with the original request
	var ctx context.Context
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})

	now := time.Now().Add(-24 * time.Hour).Round(time.Millisecond)
	req, err := http.NewRequest("GET",
		fmt.Sprintf("/api/v3/Patient/$export?_type=ExplanationOfBenefit&_since=%s&_outputFormat=ndjson&_typeFilter=ExplanationOfBenefit%%3Fservice-date%%3Dgt2001-04-01",
			now.Format(time.RFC3339Nano)),
		nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	ValidateRequestURL(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify we have the context as expected
	rp, ok := GetRequestParamsFromCtx(ctx)
	assert.True(t, ok)
	// assert.True(t, now.Equal(rp.Since), "Since parameter does not match")
	assert.Equal(t, rp.ResourceTypes, []string{"ExplanationOfBenefit"})
	assert.Equal(t, rp.Version, constants.V3Version)
	assert.Equal(t, [][]string{{"service-date", "gt2001-04-01"}}, rp.TypeFilter)
}

func TestInvalidRequestURL(t *testing.T) {

	base := "/api/v1/Patient/$export?"
	baseV3 := constants.V3Path + "Patient/$export?"
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
		{"invalidTypeFilterResourceType", fmt.Sprintf("%s_typeFilter=MedicationRequest%%3Fstatus%%3Dactive", baseV3),
			"Invalid _typeFilter Resource Type (Only EOBs valid): MedicationRequest"},
		{"invalidTypeFilterSubquery", fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Fservice-dateactive", baseV3),
			"Invalid _typeFilter parameter/value: service-dateactive"},
		{"invalidTypeFilterSubqueryParam", fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Fstatus%%3Dactive", baseV3),
			"Invalid _typeFilter subquery parameter: status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)
			req, err := http.NewRequest("GET", tt.url, nil)
			assert.NoError(t, err)

			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			ValidateRequestURL(noop).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})
	}
}

func TestValidRequestHeaders(t *testing.T) {
	ctx := context.Background()
	ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)
	req, err := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
	assert.NoError(t, err)

	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/fhir+json")
	req.Header.Set("Prefer", constants.TestRespondAsync)

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
		{"NoAcceptHeader", "", constants.TestRespondAsync, "Accept header is required"},
		{"InvalidAcceptHeader", "invalid", constants.TestRespondAsync, "application/fhir+json is the only supported response format"},
		{"NoPreferHeader", "application/fhir+json", "", "Prefer header is required"},
		{"InvalidPreferHeader", "application/fhir+json", "invalid", "Only asynchronous responses are supported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)
			req, err := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
			assert.NoError(t, err)

			req = req.WithContext(ctx)
			req.Header.Set("Accept", tt.acceptHeader)
			req.Header.Set("Prefer", tt.preferHeader)

			rr := httptest.NewRecorder()
			ValidateRequestHeaders(noop).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"v1", "/api/v1/Patient"},
		{"v2", "/api/v2/Group/all"},
		{"v3", "/api/v3/Patient/$export"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := getVersion(tt.url)

			assert.Equal(t, tt.name, version)
			assert.Equal(t, nil, err)
		})
	}
}
