package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/fhir"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			"invalidOutputFormat",
			fmt.Sprintf("%s_outputFormat=invalid", base),
			"_outputFormat parameter must be one of",
		},
		{
			"elementsNotSupported",
			fmt.Sprintf("%s_elements=invalid", base),
			"does not support the _elements parameter",
		},
		{
			"contains?",
			fmt.Sprintf("%s?_type=Patient", base),
			"query parameters cannot start with ?",
		},
		{
			"invalidSince",
			fmt.Sprintf("%s_since=05-25-1977", base),
			"Date must be in FHIR Instant format",
		},
		{
			"futureSince",
			fmt.Sprintf("%s_since=%s", base, time.Now().Add(24*time.Hour).Format(time.RFC3339Nano)),
			"Date must be a date that has already passed",
		},
		{
			"repeatedType",
			fmt.Sprintf("%s_type=Patient,Patient", base),
			"Repeated resource type Patient",
		},
		{
			"noVersion",
			"/api/Patient$export",
			"invalid version in request URL",
		},
		{
			"invalidTypeFilterResourceType",
			fmt.Sprintf("%s_typeFilter=MedicationRequest%%3Fstatus%%3Dactive", baseV3),
			"invalid _typeFilter Resource Type (Only EOBs valid): MedicationRequest",
		},
		{
			"invalidTypeFilterSubquery",
			fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Fservice-dateactive", baseV3),
			"_typeFilter value missing equals sign",
		},
		{
			"invalidTypeFilterSubqueryParam",
			fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Fstatus%%3Dactive", baseV3),
			"invalid _typeFilter subquery parameter: status",
		},
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

func TestParseHeaderValues(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		headerName string
		expected   []string
	}{
		{
			name:       "singleValue",
			headers:    http.Header{"Prefer": []string{constants.TestRespondAsync}},
			headerName: "Prefer",
			expected:   []string{constants.TestRespondAsync},
		},
		{
			name:       "commaSeparatedValues",
			headers:    http.Header{"Prefer": []string{constants.TestRespondAsync + ",handling=lenient"}},
			headerName: "Prefer",
			expected:   []string{constants.TestRespondAsync, "handling=lenient"},
		},
		{
			name:       "whitespaceTrimmingAndEmptyTokens",
			headers:    http.Header{"Prefer": []string{" " + constants.TestRespondAsync + " , , handling=lenient "}},
			headerName: "Prefer",
			expected:   []string{constants.TestRespondAsync, "handling=lenient"},
		},
		{
			name:       "multiLineHeaders",
			headers:    http.Header{"Prefer": []string{constants.TestRespondAsync, "handling=lenient"}},
			headerName: "Prefer",
			expected:   []string{constants.TestRespondAsync, "handling=lenient"},
		},
		{
			name:       "missingHeader",
			headers:    http.Header{},
			headerName: "Prefer",
			expected:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parseHeaderValues(tt.headers, tt.headerName)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestHasHeaderToken(t *testing.T) {
	tests := []struct {
		name        string
		values      []string
		targetToken string
		expected    bool
	}{
		{
			name:        "exactMatch",
			values:      []string{"respond-async"},
			targetToken: "respond-async",
			expected:    true,
		},
		{
			name:        "caseInsensitiveMatch",
			values:      []string{"RESPOND-ASYNC"},
			targetToken: "respond-async",
			expected:    true,
		},
		{
			name:        "withParameters",
			values:      []string{"respond-async; wait=10"},
			targetToken: "respond-async",
			expected:    true,
		},
		{
			name:        "mediaTypeWithParameters",
			values:      []string{"application/fhir+json; charset=utf-8"},
			targetToken: "application/fhir+json",
			expected:    true,
		},
		{
			name:        "multipleValuesMatch",
			values:      []string{"handling=lenient", "respond-async"},
			targetToken: "respond-async",
			expected:    true,
		},
		{
			name:        "noMatch",
			values:      []string{"handling=lenient"},
			targetToken: "respond-async",
			expected:    false,
		},
		{
			name:        "emptySlice",
			values:      nil,
			targetToken: "respond-async",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasHeaderToken(tt.values, tt.targetToken))
		})
	}
}

func TestValidRequestHeaders(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *http.Request
	}{
		{
			name: "StandardHeaders",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json")
				req.Header.Set("Prefer", constants.TestRespondAsync)
				return req
			},
		},
		{
			name: "PreferCommaSeparatedDirectives",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json")
				req.Header.Set("Prefer", constants.TestRespondAsync+",handling=lenient")
				return req
			},
		},
		{
			name: "PreferWithParameters",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json")
				req.Header.Set("Prefer", constants.TestRespondAsync+"; wait=10")
				return req
			},
		},
		{
			name: "CaseInsensitiveHeaders",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "APPLICATION/FHIR+JSON")
				req.Header.Set("Prefer", "RESPOND-ASYNC")
				return req
			},
		},
		{
			name: "PreferReversedDirectives",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json")
				req.Header.Set("Prefer", "handling=lenient, "+constants.TestRespondAsync)
				return req
			},
		},
		{
			name: "PreferWhitespaceTolerance",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json")
				req.Header.Set("Prefer", constants.TestRespondAsync+" , handling=lenient")
				return req
			},
		},
		{
			name: "PreferMultiLineHeaders",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json")
				req.Header.Add("Prefer", "handling=lenient")
				req.Header.Add("Prefer", constants.TestRespondAsync)
				return req
			},
		},
		{
			name: "AcceptCommaSeparatedMediaTypes",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json, application/json")
				req.Header.Set("Prefer", constants.TestRespondAsync)
				return req
			},
		},
		{
			name: "AcceptWithParameters",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Set("Accept", "application/fhir+json; charset=utf-8")
				req.Header.Set("Prefer", constants.TestRespondAsync)
				return req
			},
		},
		{
			name: "AcceptMultiLineHeaders",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
				req.Header.Add("Accept", "application/json")
				req.Header.Add("Accept", "application/fhir+json")
				req.Header.Set("Prefer", constants.TestRespondAsync)
				return req
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)
			req := tt.setupReq().WithContext(ctx)

			rr := httptest.NewRecorder()
			ValidateRequestHeaders(noop).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
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
		{"UnsupportedAcceptHeader", "application/xml", constants.TestRespondAsync, "application/fhir+json is the only supported response format"},
		{"NoPreferHeader", "application/fhir+json", "", "Prefer header is required"},
		{"InvalidPreferHeader", "application/fhir+json", "invalid", "Only asynchronous responses are supported"},
		{"PreferMissingRespondAsync", "application/fhir+json", "handling=lenient", "Only asynchronous responses are supported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)
			req, err := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
			assert.NoError(t, err)

			req = req.WithContext(ctx)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			if tt.preferHeader != "" {
				req.Header.Set("Prefer", tt.preferHeader)
			}

			rr := httptest.NewRecorder()
			ValidateRequestHeaders(noop).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.errMsg)
		})
	}
}

func TestValidateTypeFilterTagCodes(t *testing.T) {
	baseV3 := constants.V3Path + "Patient/$export?"
	ctx := context.Background()
	ctx = log.NewStructuredLoggerEntry(logrus.New(), ctx)

	tests := []struct {
		name               string
		url                string
		shouldFail         bool
		expectedErr        error
		description        string
		expectedTypeFilter fhir.TypeFilterSubquery // when non-nil, we assert the parsed TypeFilter in context equals this making sure params are not dropped.
	}{
		{
			name:        "validTagSharedSystem",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CSharedSystem", baseV3),
			shouldFail:  false,
			description: "Valid tag in URL format should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDSystemTypeURL + "|SharedSystem",
					},
				},
			},
		},
		{
			name:        "validTagNationalClaimsHistory",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CNationalClaimsHistory", baseV3),
			shouldFail:  false,
			description: "Valid NCH tag should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDSystemTypeURL + "|NationalClaimsHistory",
					},
				},
			},
		},
		{
			name:        "validTagSharedSystemComma",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CSharedSystem,https%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CFinalAction", baseV3),
			shouldFail:  false,
			description: "Valid comma-separated tags should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDSystemTypeURL + "|SharedSystem," + constants.BFDFinalActionURL + "|FinalAction",
					},
				},
			},
		},
		{
			name:        "validOutcomeComma",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Foutcome%%3Dpartial,complete", baseV3),
			shouldFail:  false,
			description: "Valid comma-separated outcome should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "outcome",
						Value: "partial,complete",
					},
				},
			},
		},
		{
			name:        "invalidOutcome",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Foutcome%%3Dinvalid_status", baseV3),
			shouldFail:  true,
			expectedErr: fhir.ParameterValidationError{},
			description: "Outcome must be complete or partial",
		},
		{
			name:        "validTagDDPS",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CDDPS", baseV3),
			shouldFail:  false,
			description: "Valid DDPS tag should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDSystemTypeURL + "|DDPS",
					},
				},
			},
		},
		{
			name:        "validTagFinalAction",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CFinalAction", baseV3),
			shouldFail:  false,
			description: "Valid FinalAction tag should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDFinalActionURL + "|FinalAction",
					},
				},
			},
		},
		{
			name:        "validTagNotFinalAction",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction", baseV3),
			shouldFail:  false,
			description: "Valid NotFinalAction tag should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDFinalActionURL + "|NotFinalAction",
					},
				},
			},
		},
		{
			name:        "invalidTagPartiallyAdjudicated",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3DPartiallyAdjudicated", baseV3),
			shouldFail:  true,
			expectedErr: fhir.ParameterValidationError{},
			description: "Old PartiallyAdjudicated tag should be rejected",
		},
		{
			name:        "invalidTagSharedSystem",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3DSharedSystem", baseV3),
			shouldFail:  true,
			expectedErr: fhir.ParameterValidationError{},
			description: "Only code, no system should be rejected. even with valid code",
		},
		{
			name:        "invalidTagRandomValue",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3DInvalidTag", baseV3),
			shouldFail:  true,
			expectedErr: fhir.ParameterValidationError{},
			description: "Random invalid tag should be rejected",
		},
		{
			name:        "multipleValidTags",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction%%26_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CSharedSystem", baseV3),
			shouldFail:  false,
			description: "Multiple valid tags should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "_tag",
						Value: constants.BFDFinalActionURL + "|NotFinalAction",
					},
					{
						Name:  "_tag",
						Value: constants.BFDSystemTypeURL + "|SharedSystem",
					},
				},
			},
		},
		{
			name:        "validTypeFilterServiceDateAndTag",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Fservice-date%%3Dlt2021-02-15%%26_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CFinalAction", baseV3),
			shouldFail:  false,
			description: "Subquery with service-date and _tag (FinalAction) should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "service-date",
						Value: "lt2021-02-15",
					},
					{
						Name:  "_tag",
						Value: constants.BFDFinalActionURL + "|FinalAction",
					},
				},
			},
		},
		{
			name:        "validTypeFilterServiceDateAndTagAlternate",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3Fservice-date%%3Dgt2001-04-01%%26_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FSystem-Type%%7CNationalClaimsHistory", baseV3),
			shouldFail:  false,
			description: "Subquery with service-date and _tag (NationalClaimsHistory) should pass",
			expectedTypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "ExplanationOfBenefit",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "service-date",
						Value: "gt2001-04-01",
					},
					{
						Name:  "_tag",
						Value: constants.BFDSystemTypeURL + "|NationalClaimsHistory",
					},
				},
			},
		},
		{
			name:        "multipleTagsOneInvalid",
			url:         fmt.Sprintf("%s_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction%%26_tag%%3DPartiallyAdjudicated", baseV3),
			shouldFail:  true,
			expectedErr: fhir.ParameterValidationError{},
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

		{
			name:        "multipleTypeFilterSubqueries",
			url:         fmt.Sprintf("%s_type=ExplanationOfBenefit&_typeFilter=ExplanationOfBenefit%%3Fservice-date%%3Dlt2021-02-15%%26_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CFinalAction&_typeFilter=ExplanationOfBenefit%%3F_tag%%3Dhttps%%3A%%2F%%2Fbluebutton.cms.gov%%2Ffhir%%2FCodeSystem%%2FFinal-Action%%7CNotFinalAction", baseV3),
			shouldFail:  true,
			expectedErr: fhir.ParameterValidationError{},
			description: "Multiple _typeFilter params. Bulk IG supports this but we do not",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			assert.NoError(t, err)
			req = req.WithContext(ctx)
			req.Header.Set("Accept", "application/fhir+json")
			req.Header.Set("Prefer", constants.TestRespondAsync)

			var capturedCtx context.Context
			handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				capturedCtx = r.Context()
				rw.WriteHeader(http.StatusOK)
			})

			rr := httptest.NewRecorder()
			ValidateRequestURL(handler).ServeHTTP(rr, req)

			if tt.shouldFail {
				assert.Equal(t, http.StatusBadRequest, rr.Code, tt.description)
				assert.Contains(t, rr.Body.String(), tt.expectedErr.Error(), tt.description)
			} else {
				assert.Equal(t, http.StatusOK, rr.Code, tt.description)
				if tt.expectedTypeFilter.QueryParameters != nil {
					rp, ok := GetRequestParamsFromCtx(capturedCtx)
					assert.True(t, ok, "request params should be in context")
					assert.Equal(t, tt.expectedTypeFilter, rp.TypeFilter, "parsed _typeFilter params should match request")
				}
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

func (s *MiddlewareTestSuite) TestValidateShared() {
	resp := httptest.NewRecorder()

	body := strings.NewReader(`
		{
			"system": "test-system",
			"version": "v3",
			"tracking_id": "test-tracking-id",
			"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
			"queries": [
				{
					"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
					"resource_types": ["Patient"],
					"headers": {
						"Content-Type": "application/json"
					},
					"form_data": [
						{
							"name": "_since",
							"value": "2026-08-01T08:00:00-05:00"
						},
						{
							"name": "_typeFilter",
							"value": {
								"resource_type": "ExplanationOfBenefit",
								"query_parameters": [
									{
										"name": "service-date",
										"value": "lt2021-02-15"
									},
									{
										"name": "_tag",
										"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
									}
								]
							}
						}
					]
				},
				{
					"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
					"resource_types": ["Claim", "ClaimResponse"],
					"form_data": [
						{
							"name": "_since",
							"value": "2023-08-01T08:00:00-05:00"
						}
					]
				}
			]
		}
	`)

	// create a handler to verify data set into context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries, ok := GetSharedExportQueriesFromCtx(r.Context())
		require.True(s.T(), ok)
		require.NotNil(s.T(), queries)
		require.Len(s.T(), queries, 2)

		// validate the contents of the queries
		assert.Equal(s.T(), "https://test.test.cms.gov/v3/Patient&params=test", queries[0].SourceQuery)
		assert.Equal(s.T(), "test-system", queries[0].System)
		assert.Equal(s.T(), "test-tracking-id", queries[0].TrackingID)
		assert.Equal(s.T(), "v3", queries[0].Version)
		assert.Equal(s.T(), []string{"1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"}, queries[0].MBIs)
		assert.Equal(s.T(), []string{"Patient"}, queries[0].ResourceTypes)
		assert.Equal(s.T(), map[string]string{"Content-Type": "application/json"}, queries[0].Headers)
		assert.Equal(s.T(), time.Date(2026, 8, 1, 8, 0, 0, 0, time.FixedZone("", -5*60*60)), queries[0].Since)
		assert.Equal(s.T(), "ExplanationOfBenefit", queries[0].TypeFilter.ResourceType)
		require.Equal(s.T(), 2, len(queries[0].TypeFilter.QueryParameters))
		assert.Equal(s.T(), "service-date", queries[0].TypeFilter.QueryParameters[0].Name)
		assert.Equal(s.T(), "lt2021-02-15", queries[0].TypeFilter.QueryParameters[0].Value)
		assert.Equal(s.T(), "_tag", queries[0].TypeFilter.QueryParameters[1].Name)
		assert.Equal(s.T(), "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction", queries[0].TypeFilter.QueryParameters[1].Value)

		assert.Equal(s.T(), []string{"1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"}, queries[1].MBIs)
		assert.Equal(s.T(), []string{"Claim", "ClaimResponse"}, queries[1].ResourceTypes)
		assert.Equal(s.T(), time.Date(2023, 8, 1, 8, 0, 0, 0, time.FixedZone("", -5*60*60)), queries[1].Since)
	})

	handlerToTest := ValidateSharedExportPayload(nextHandler)
	req := httptest.NewRequest("GET", "http:///v1/testing", body)
	handlerToTest.ServeHTTP(resp, req)
	require.Equal(s.T(), 200, resp.Code)
}

func (s *MiddlewareTestSuite) TestValidateShared_OnlyRequiredFields() {
	resp := httptest.NewRecorder()

	body := strings.NewReader(`
		{
			"system": "test-system",
			"queries": [
				{
					"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
					"resource_types": ["Claim", "ClaimResponse"],
					"form_data": [
						{
							"name": "_since",
							"value": "2023-08-01T08:00:00-05:00"
						}
					]
				}
			]
		}
	`)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries, ok := GetSharedExportQueriesFromCtx(r.Context())
		require.True(s.T(), ok)
		require.NotNil(s.T(), queries)
		require.Len(s.T(), queries, 1)
	})

	handlerToTest := ValidateSharedExportPayload(nextHandler)
	req := httptest.NewRequest("GET", "http:///v1/testing", body)
	handlerToTest.ServeHTTP(resp, req)
	require.Equal(s.T(), 200, resp.Code)
}

func (s *MiddlewareTestSuite) TestValidateShared_InvalidJSON() {
	resp := httptest.NewRecorder()

	body := strings.NewReader(`
		{
			"system": "test-system",
			"bad"
		}
	`)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})

	handlerToTest := ValidateSharedExportPayload(nextHandler)
	req := httptest.NewRequest("GET", "http:///v1/testing", body)
	handlerToTest.ServeHTTP(resp, req)
	require.Equal(s.T(), 400, resp.Code)
}

func (s *MiddlewareTestSuite) TestValidateShared_InvalidData() {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name string
		body string // to note Ive removed the relevant field from the second query in each of the below cases
	}{
		{
			"missing queries",
			`{
				"system": "test-system",
				"tracking_id": "test-tracking-id",
				"version": "v3",
				"queries": []
			}`,
		}, {
			"missing mbis",
			`{
				"system": "test-system",
				"tracking_id": "test-tracking-id",
				"version": "v3",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "2026-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"resource_types": ["Claim", "ClaimResponse"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}
						]
					}
				]
			}`,
		}, {
			"missing resource_types",
			`{
				"system": "test-system",
				"tracking_id": "test-tracking-id",
				"version": "v3",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "2026-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}
						]
					}
				]
			}`,
		}, {
			"missing _since",
			`{
				"system": "test-system",
				"tracking_id": "test-tracking-id",
				"version": "v3",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Claim", "ClaimResponse"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": []
					}
				]
			}`,
		}, {
			"invalid _since",
			`{
				"system": "test-system",
				"tracking_id": "test-tracking-id",
				"version": "v3",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Claim", "ClaimResponse"],
						"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
						"form_data": [
							{
								"name": "_since",
								"value": "bad-date-time"
							}
						]
					}
				]
			}`,
		}, {
			"invalid _typeFilter value",
			`{
				"system": "test-system",
				"version": "v3",
				"tracking_id": "test-tracking-id",
				"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"form_data": [
							{
								"name": "_since",
								"value": "2026-08-01T08:00:00-05:00"
							},
							{
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Claim", "ClaimResponse"],
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": "bad-type-filter"
							}
						]
					}
				]
			}`,
		}, {
			"invalid _typeFilter sub query params",
			`{
				"system": "test-system",
				"version": "v3",
				"tracking_id": "test-tracking-id",
				"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"form_data": [
							{
								"name": "_since",
								"value": "2026-08-01T08:00:00-05:00"
							},
							{
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Claim", "ClaimResponse"],
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "bad-service-date"
										},
										{
											"name": "_tag",
											"value": "bad-tag"
										}
									]
								}
							}
						]
					}
				]
			}`,
		}, {
			"invalid _typeFilter resource type",
			`{
				"system": "test-system",
				"version": "v3",
				"tracking_id": "test-tracking-id",
				"source_query": "https://test.test.cms.gov/v3/Patient&params=test",
				"queries": [
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Patient"],
						"form_data": [
							{
								"name": "_since",
								"value": "2026-08-01T08:00:00-05:00"
							},
							{
								"name": "_typeFilter",
								"value": {
									"resource_type": "ExplanationOfBenefit",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					},
					{
						"mbis": ["1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"],
						"resource_types": ["Claim", "ClaimResponse"],
						"form_data": [
							{
								"name": "_since",
								"value": "2023-08-01T08:00:00-05:00"
							}, {
								"name": "_typeFilter",
								"value": {
									"resource_type": "Patient",
									"query_parameters": [
										{
											"name": "service-date",
											"value": "lt2021-02-15"
										},
										{
											"name": "_tag",
											"value": "https://bluebutton.cms.gov/fhir/CodeSystem/Final-Action|FinalAction"
										}
									]
								}
							}
						]
					}
				]
			}`,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "http:///v1/testing", strings.NewReader(tt.body))

			ValidateSharedExportPayload(nextHandler).ServeHTTP(resp, req)
			assert.Equal(s.T(), http.StatusBadRequest, resp.Code)
			assert.Contains(s.T(), resp.Body.String(), "invalid params")
		})
	}
}
