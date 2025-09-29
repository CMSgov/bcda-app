package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	responseutilsv2 "github.com/CMSgov/bcda-app/bcda/responseutils/v2"
	"github.com/CMSgov/bcda-app/log"
)

var supportedOutputFormats = map[string]struct{}{
	"ndjson":                  {},
	"application/fhir+ndjson": {},
	"application/ndjson":      {}}

type RequestParameters struct {
	Since         time.Time
	ResourceTypes []string
	Version       string // e.g. v1, v2
	RequestURL    string
	TypeFilter    [][]string
}

// const BBSystemURL = "https://bluebutton.cms.gov/fhir/CodeSystem/Adjudication-Status"

// requestkey is an unexported context key to avoid collisions
type requestkey int

const rk requestkey = 0

// TODO: replace this function else where with line 35
func SetRequestParamsCtx(ctx context.Context, rp RequestParameters) context.Context {
	return context.WithValue(ctx, rk, rp)
}

func GetRequestParamsFromCtx(ctx context.Context) (RequestParameters, bool) {
	rp, ok := ctx.Value(rk).(RequestParameters)
	return rp, ok
}

func validateOutputFormat(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) bool {
	params, ok := r.URL.Query()["_outputFormat"]
	if !ok {
		return true
	}

	if _, found := supportedOutputFormats[params[0]]; !found {
		errMsg := fmt.Sprintf("_outputFormat parameter must be one of %v", getKeys(supportedOutputFormats))
		log.API.Error(errMsg)
		rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
		return false
	}
	return true
}

// we do not support "_elements" parameter
func validateElementsParameter(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) bool {
	_, ok := r.URL.Query()["_elements"]
	if !ok {
		return true
	}

	errMsg := "Invalid parameter: this server does not support the _elements parameter."
	log.API.Warn(errMsg)
	rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
	return false
}

// Check and see if the user has a duplicated the query parameter symbol (?)
// e.g. /api/v1/Patient/$export?_type=ExplanationOfBenefit&?_since=2020-09-13T08:00:00.000-05:00
func validateQueryParameterFormat(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) bool {
	for key := range r.URL.Query() {
		if strings.HasPrefix(key, "?") {
			errMsg := "Invalid parameter: query parameters cannot start with ?"
			log.API.Warn(errMsg)
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
			return false
		}
	}
	return true
}

func validateSinceParameter(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) (time.Time, bool) {
	params, ok := r.URL.Query()["_since"]
	if !ok {
		return time.Time{}, true
	}

	sinceDate, err := time.Parse(time.RFC3339Nano, params[0])
	if err != nil {
		errMsg := "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."
		log.API.Warn(errMsg)
		rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
		return time.Time{}, false
	}

	if sinceDate.After(time.Now()) {
		errMsg := "Invalid date format supplied in _since parameter. Date must be a date that has already passed"
		log.API.Warn(errMsg)
		rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
		return time.Time{}, false
	}

	return sinceDate, true
}

func validateResourceTypes(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) ([]string, bool) {
	params, ok := r.URL.Query()["_type"]
	if !ok {
		return nil, true
	}

	// validate no duplicate resource types
	resourceMap := make(map[string]struct{})
	resourceTypes := strings.Split(params[0], ",")
	for _, resource := range resourceTypes {
		if _, ok := resourceMap[resource]; !ok {
			resourceMap[resource] = struct{}{}
		} else {
			errMsg := fmt.Sprintf("Repeated resource type %s", resource)
			log.API.Error(errMsg)
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
			return nil, false
		}
	}
	return resourceTypes, true
}

func validateTypeFilterParameter(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter, version string) ([][]string, bool) {
	params, ok := r.URL.Query()["_typeFilter"]
	if version != "v3" || !ok {
		return nil, true
	}

	var typeFilterParams [][]string
	for _, subQuery := range params {
		// The subquery is url-encoded. So we will first decode so we can parse it
		decodedQuery, err := url.QueryUnescape(subQuery)
		if err != nil {
			errMsg := fmt.Sprintf("failed to unescape %s", subQuery)
			log.API.Error(errMsg)
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
			return nil, false
		}

		// Expected format is: <resourceType>?<paramList>
		resourceType, queryParams, ok := strings.Cut(decodedQuery, "?")
		if !ok {
			errMsg := fmt.Sprintf("missing question mark %s", decodedQuery)
			log.API.Error(errMsg)
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
			return nil, false
		}

		// Right now, we are only accepting ExplanationOfBenefit subqueries
		if resourceType != "ExplanationOfBenefit" {
			errMsg := fmt.Sprintf("Invalid _typeFilter Resource Type (Only EOBs valid): %s", resourceType)
			log.API.Error(errMsg)
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
			return nil, false
		}

		// Loop through the param list from the subquery
		paramAry := strings.Split(queryParams, "&")
		for _, paramPair := range paramAry {
			paramName, paramValue, ok := strings.Cut(paramPair, "=")
			if !ok {
				errMsg := fmt.Sprintf("Invalid _typeFilter parameter/value: %s", paramPair)
				log.API.Error(errMsg)
				rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
				return nil, false
			}

			if slices.Contains([]string{"service-date", "_tag", "_profile"}, paramName) {
				typeFilterParams = append(typeFilterParams, []string{paramName, paramValue})
			} else {
				errMsg := fmt.Sprintf("Invalid _typeFilter subquery parameter: %s", paramName)
				log.API.Error(errMsg)
				rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
				return nil, false
			}
		}
	}
	return typeFilterParams, true
}

// ValidateRequestURL ensure that request matches certain expectations.
// Any error that it finds will result in a http.StatusBadRequest response.
// If successful, it populates the request context with RequestParameters that can be used downstream.
// These paramters can be retrieved by calling RequestParametersFromContext.
func ValidateRequestURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw, version := getResponseWriterFromRequestPath(w, r)
		if rw == nil {
			return
		}

		// Validate all parameters
		if !validateOutputFormat(r, rw, w) ||
			!validateElementsParameter(r, rw, w) ||
			!validateQueryParameterFormat(r, rw, w) {
			return
		}

		// Validate _since parameter
		sinceDate, valid := validateSinceParameter(r, rw, w)
		if !valid {
			return
		}

		// Validate resource types
		resourceTypes, valid := validateResourceTypes(r, rw, w)
		if !valid {
			return
		}

		// Validate type filter for v3
		typeFilter, valid := validateTypeFilterParameter(r, rw, w, version)
		if !valid {
			return
		}

		// Build request parameters
		rp := RequestParameters{
			Version:       version,
			RequestURL:    r.URL.String(),
			Since:         sinceDate,
			ResourceTypes: resourceTypes,
			TypeFilter:    typeFilter,
		}

		ctx := SetRequestParamsCtx(r.Context(), rp)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ValidateRequestHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header

		acceptHeader := h.Get("Accept")
		preferHeader := h.Get("Prefer")

		logger := log.GetCtxLogger(r.Context())

		rw, _ := getResponseWriterFromRequestPath(w, r)
		if rw == nil {
			return
		}

		if acceptHeader == "" {
			logger.Warn("Accept header is required")
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "Accept header is required")
			return
		} else if acceptHeader != "application/fhir+json" {
			logger.Warn("application/fhir+json is the only supported response format")
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "application/fhir+json is the only supported response format")
			return
		}

		if preferHeader == "" {
			logger.Warn("Prefer header is required")
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "Prefer header is required")
			return
		} else if preferHeader != "respond-async" {
			logger.Warn("Only asynchronous responses are supported")
			rw.Exception(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "Only asynchronous responses are supported")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getKeys(kv map[string]struct{}) []string {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	return keys
}

var versionExp = regexp.MustCompile(`\/api\/(v\d+)\/`)

func getVersion(path string) (string, error) {
	parts := versionExp.FindStringSubmatch(path)
	if len(parts) != 2 {
		return "", fmt.Errorf("cannot retrieve version: not enough parts in path")
	}
	return parts[1], nil
}

type fhirResponseWriter interface {
	Exception(context.Context, http.ResponseWriter, int, string, string)
	NotFound(context.Context, http.ResponseWriter, int, string, string)
}

func getRespWriter(version string) (fhirResponseWriter, error) {
	switch version {
	case "v1":
		return responseutils.NewFhirResponseWriter(), nil
	case "v2":
		return responseutilsv2.NewFhirResponseWriter(), nil
	case constants.V3Version:
		return responseutilsv2.NewFhirResponseWriter(), nil // TODO: V3
	default:
		return nil, fmt.Errorf("unexpected API version: %s", version)
	}
}

func getResponseWriterFromRequestPath(w http.ResponseWriter, r *http.Request) (fhirResponseWriter, string) {
	version, err := getVersion(r.URL.Path)
	if err != nil {
		logger := log.GetCtxLogger(r.Context())
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, ""
	}
	rw, err := getRespWriter(version)
	if err != nil {
		logger := log.GetCtxLogger(r.Context())
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, ""
	}

	return rw, version
}
