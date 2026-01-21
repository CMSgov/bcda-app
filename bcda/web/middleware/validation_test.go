package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
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

func TestValidateTagSubqueryParameter(t *testing.T) {
	tests := []struct {
		name     string
		tagValue string
		expected bool
	}{
		{"codeOnly", "SharedSystem", false},
		{"invalidCode", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|12345", false},
		{"invalidSystem", "https://bluebutton.cms.gov/fhir/CodeSystem/12345|FinalAction", false},
		{"codeDoesNotMatchSystem", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NotFinalAction", false},
		{"validSystemAndCode", "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory", true},
		{"emptyString", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := ValidateTagSubqueryParameter(tt.tagValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateTypeFilterTagCodes(t *testing.T) {
	baseV3 := constants.V3Path + "Patient/$export?"
	ctx := context.Background()
	ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)

	tests := []struct {
		name        string
		url         string
		shouldFail  bool
		errMsg      string
		description string
	}{
		{
			name:        "validTagSharedSystem",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CSharedSystem", baseV3),
			shouldFail:  false,
			description: "Valid tag in URL format should pass",
		},
		{
			name:        "validTagNationalClaimsHistory",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CNationalClaimsHistory", baseV3),
			shouldFail:  false,
			description: "Valid NotFinalAction tag should pass",
		},
		{
			name:        "validTagFinalAction",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CFinalAction", baseV3),
			shouldFail:  false,
			description: "Valid FinalAction tag should pass",
		},
		{
			name:        "validTagNotFinalAction",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction", baseV3),
			shouldFail:  false,
			description: "Valid NationalClaimsHistory tag should pass",
		},
		{
			name:        "invalidTagPartiallyAdjudicated",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3DPartiallyAdjudicated", baseV3),
			shouldFail:  true,
			errMsg:      "Invalid _tag value: PartiallyAdjudicated. Searching by tag requires a token (system|code) to be specified",
			description: "Old PartiallyAdjudicated tag should be rejected",
		},
		{
			name:        "invalidTagSharedSystem",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3DSharedSystem", baseV3),
			shouldFail:  true,
			errMsg:      "Invalid _tag value: SharedSystem. Searching by tag requires a token (system|code) to be specified",
			description: "Only code, no system should be rejected. even with valid code",
		},
		{
			name:        "invalidTagRandomValue",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3DInvalidTag", baseV3),
			shouldFail:  true,
			errMsg:      "Invalid _tag value: InvalidTag. Searching by tag requires a token (system|code) to be specified",
			description: "Random invalid tag should be rejected",
		},
		{
			name:        "multipleValidTags",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction%%26_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CSharedSystem", baseV3),
			shouldFail:  false,
			description: "Multiple valid tags should pass",
		},
		{
			name:        "multipleTagsOneInvalid",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction%%26_tag%%3DPartiallyAdjudicated", baseV3),
			shouldFail:  true,
			errMsg:      "Invalid _tag value: PartiallyAdjudicated",
			description: "Multiple tags with one invalid should fail",
		},
		{
			name:        "v2ShouldIgnoreTypeFilter",
			url:         fmt.Sprintf("/api/v2/Patient/$export?_typeFilter=ExplanationOfBenefit%%3F_tag%%3DPartiallyAdjudicated"),
			shouldFail:  false,
			description: "v2 should ignore _typeFilter validation (old behavior preserved)",
		},
		{
			name:        "v1ShouldIgnoreTypeFilter",
			url:         fmt.Sprintf("/api/v1/Patient/$export?_typeFilter=ExplanationOfBenefit%%3F_tag%%3DPartiallyAdjudicated"),
			shouldFail:  false,
			description: "v1 should ignore _typeFilter validation (old behavior preserved)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			assert.NoError(t, err)
			req = req.WithContext(ctx)
			req.Header.Set("Accept", "application/fhir+json")
			req.Header.Set("Prefer", constants.TestRespondAsync)

			rr := httptest.NewRecorder()
			ValidateRequestURL(noop).ServeHTTP(rr, req)

			if tt.shouldFail {
				assert.Equal(t, http.StatusBadRequest, rr.Code, tt.description)
				assert.Contains(t, rr.Body.String(), tt.errMsg, tt.description)
			} else {
				assert.Equal(t, http.StatusOK, rr.Code, tt.description)
			}
		})
	}
}

func TestGetRespWriter(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		path string
	}{
		{"v1", constants.V1Path},
		{"v2", constants.V2Path},
		{"v3", constants.V3Path},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			rw := auth.GetRespWriter(tt.path)
			rw.OpOutcome(ctx, rr, http.StatusUnauthorized, responseutils.TokenErr, responseutils.TokenErr)
			resp := rr.Body.String()
			switch tt.name {
			case "v1":
				assert.NotContains(t, resp, "coding")
			case "v2":
				assert.Contains(t, resp, "coding")
			case "v3":
				assert.NotContains(t, resp, "coding")
			}
		})
	}
}
