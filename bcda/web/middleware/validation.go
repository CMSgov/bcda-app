package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	rt "github.com/CMSgov/bcda-app/bcda/api/request_types"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/fhir"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	responseutilsv2 "github.com/CMSgov/bcda-app/bcda/responseutils/v2"
	responseutilsv3 "github.com/CMSgov/bcda-app/bcda/responseutils/v3"
	"github.com/CMSgov/bcda-app/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var supportedOutputFormats = map[string]struct{}{
	"ndjson":                  {},
	"application/fhir+ndjson": {},
	"application/ndjson":      {}}

// requestkey is an unexported context key to avoid collisions
type requestkey int

const rk requestkey = 0

type ctxSharedExportQueriesKey struct{}

func GetSharedExportQueriesFromCtx(ctx context.Context) ([]rt.SharedExportQuery, bool) {
	queries, ok := ctx.Value(ctxSharedExportQueriesKey{}).([]rt.SharedExportQuery) // Type assertion
	return queries, ok
}

// TODO: replace this function else where with line 35
func SetRequestParamsCtx(ctx context.Context, rp rt.RequestParameters) context.Context {
	return context.WithValue(ctx, rk, rp)
}

func GetRequestParamsFromCtx(ctx context.Context) (rt.RequestParameters, bool) {
	rp, ok := ctx.Value(rk).(rt.RequestParameters)
	return rp, ok
}

func validateOutputFormat(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) bool {
	ctx := r.Context()
	params, ok := r.URL.Query()["_outputFormat"]
	if !ok {
		return true
	}

	if _, found := supportedOutputFormats[params[0]]; !found {
		errMsg := fmt.Sprintf("_outputFormat parameter must be one of %v", getKeys(supportedOutputFormats))
		ctx, _ = log.WriteWarnWithFields(
			ctx,
			fmt.Sprintf("%s: %s", responseutils.FormatErr, errMsg),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
		return false
	}
	return true
}

// we do not support "_elements" parameter
func validateElementsParameter(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) bool {
	_, ok := r.URL.Query()["_elements"]
	ctx := r.Context()
	if !ok {
		return true
	}

	errMsg := "Invalid parameter: this server does not support the _elements parameter."
	ctx, _ = log.WriteWarnWithFields(
		ctx,
		fmt.Sprintf("%s: %s", responseutils.RequestErr, errMsg),
		logrus.Fields{"resp_status": http.StatusBadRequest},
	)
	rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
	return false
}

// Check and see if the user has a duplicated the query parameter symbol (?)
// e.g. /api/v1/Patient/$export?_type=ExplanationOfBenefit&?_since=2020-09-13T08:00:00.000-05:00
func validateQueryParameterFormat(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) bool {
	ctx := r.Context()

	for key := range r.URL.Query() {
		if strings.HasPrefix(key, "?") {
			errMsg := "Invalid parameter: query parameters cannot start with ?"
			ctx, _ = log.WriteWarnWithFields(
				ctx,
				fmt.Sprintf("%s: %s", responseutils.FormatErr, errMsg),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
			return false
		}
	}
	return true
}

func validateSinceParameter(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) (time.Time, bool) {
	ctx := r.Context()
	params, ok := r.URL.Query()["_since"]
	if !ok {
		return time.Time{}, true
	}

	sinceDate, err := time.Parse(time.RFC3339Nano, params[0])
	if err != nil {
		errMsg := "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format."
		ctx, _ = log.WriteWarnWithFields(
			ctx,
			fmt.Sprintf("%s: %s", responseutils.FormatErr, errMsg),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
		return time.Time{}, false
	}

	if sinceDate.After(time.Now()) {
		errMsg := "Invalid date format supplied in _since parameter. Date must be a date that has already passed"
		ctx, _ = log.WriteWarnWithFields(
			ctx,
			fmt.Sprintf("%s: %s", responseutils.FormatErr, errMsg),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, errMsg)
		return time.Time{}, false
	}

	return sinceDate, true
}

func validateResourceTypes(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter) ([]string, bool) {
	ctx := r.Context()
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
			ctx, _ = log.WriteWarnWithFields(
				ctx,
				fmt.Sprintf("%s: %s", responseutils.RequestErr, errMsg),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
			return nil, false
		}
	}
	return resourceTypes, true
}

// validateTypeFilterParameter validates the contents of the typeFilter param.
func validateTypeFilterParameter(r *http.Request, rw fhirResponseWriter, w http.ResponseWriter, version string) (fhir.TypeFilterSubquery, bool) {
	var typeFilterParam fhir.TypeFilterSubquery
	ctx := r.Context()

	params, ok := r.URL.Query()["_typeFilter"]
	if version != constants.V3Version || !ok {
		return typeFilterParam, true
	}

	typeFilterParams, err := GetTypeFilterSubquery(params)
	if err != nil {
		ctx, _ = log.WriteWarnWithFields(
			ctx,
			fmt.Sprintf("%s: %s", responseutils.RequestErr, err.Error()),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return typeFilterParam, false
	}
	return typeFilterParams, true
}

// TODO: BCDA-10505 validate the request body, headers, etc
// separate ticket: bcda api checks the aco config several times, we should add this to request context so we don't need to regex+loop each time
// ValidateSharedExportPayload validates the payload, converts to queries, and sets them into context
func ValidateSharedExportPayload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rw, err := GetRespWriter(r.URL.Path)
		if err != nil {
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "invalid version in request URL")
			return
		}

		var payload rt.SharedExportPayload
		err = json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			log.API.Error("invalid JSON payload, err: %s", err.Error())
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "invalid JSON payload")
			return
		}
		defer r.Body.Close()

		queries, invalidResults := validateSharedExportPayload(payload)
		if invalidResults != "" {
			log.API.Warn("invalid params: " + invalidResults)
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "invalid params: "+invalidResults)
			return
		}

		ctx := context.WithValue(r.Context(), ctxSharedExportQueriesKey{}, queries)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateSharedExportPayload(payload rt.SharedExportPayload) ([]rt.SharedExportQuery, string) {
	fmt.Printf("----- payload: %+v\n", payload)
	if len(payload.Queries) == 0 {
		return []rt.SharedExportQuery{}, "no queries"
	}

	issues := []string{}
	queries := []rt.SharedExportQuery{}
	for _, query := range payload.Queries {
		if len(query.DataTypes) == 0 {
			issues = append(issues, "missing data_types in query")
		}
		if len(query.MBIs) == 0 {
			issues = append(issues, "missing mbis in query")
		}
		if len(query.ResourceTypes) == 0 {
			issues = append(issues, "missing resource_types in query")
		}
		if query.SourceQuery == "" {
			issues = append(issues, "missing source_query in query")
		}
		if query.ParsedQuery.Parameters.Since == "" {
			issues = append(issues, "missing parsed_query.parameters._since in query")
		}
		_, err := time.Parse(time.RFC3339Nano, query.ParsedQuery.Parameters.Since)
		if err != nil {
			issues = append(issues, "_since must be in RFC3339")
		}
		// TODO these might not be required, and TypeFilter is being modified in the data contract
		// if len(query.ParsedQuery.Headers) == 0 {
		// 	issues = append(issues, "missing parsed_query.headers in query")
		// }
		// if query.ParsedQuery.Parameters.TypeFilter == "" {
		// 	issues = append(issues, "missing parsed_query.parameters._since in query")
		// }
		if query.Version == "" {
			issues = append(issues, "missing version in query")
		}

		queries = append(queries, query)
	}

	return queries, strings.Join(issues, "; ")
}

// GetTypeFilterSubquery parses the _typeFilter subquery
// For _tag, it validates each comma-separated token to correctly resolve compound query filters.
func GetTypeFilterSubquery(params []string) (fhir.TypeFilterSubquery, error) {
	var subquery fhir.TypeFilterSubquery

	// If more than one _typeFilter param (a logical "or"), return an error, we do not support that yet
	if len(params) > 1 {
		return subquery, fhir.ParameterValidationError{Details: "failed to process request given more that one _typeFilter parameter"}
	}

	subquery, err := fhir.ParseTypeFilterSubquery(params[0])
	if err != nil {
		return subquery, err
	}

	err = fhir.ValidateTypeFilterSubquery(subquery)
	if err != nil {
		return subquery, err
	}

	return subquery, nil
}

func HasSystemTypeTag(subquery fhir.TypeFilterSubquery) bool {
	tagParams, _ := fhir.GetTagParams(subquery)
	for _, tagParam := range tagParams {
		for _, tagValue := range tagParam.Values {
			if tagValue.System == constants.BFDSystemTypeURL {
				return true
			}
		}
	}
	return false
}

// ValidateRequestURL ensure that request matches certain expectations.
// Any error that it finds will result in a http.StatusBadRequest response.
// If successful, it populates the request context with rt.RequestParameters that can be used downstream.
// These paramters can be retrieved by calling rt.RequestParametersFromContext.
func ValidateRequestURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw, err := GetRespWriter(r.URL.Path)
		if err != nil {
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "invalid version in request URL")
			return
		}

		v, err := getVersion(r.URL.Path)
		if err != nil {
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
		typeFilter, valid := validateTypeFilterParameter(r, rw, w, v)
		if !valid {
			return
		}

		// Build request parameters
		rp := rt.RequestParameters{
			Version:       v,
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
		ctx := r.Context()
		h := r.Header

		rw, err := GetRespWriter(r.URL.Path)
		if err != nil {
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, "invalid version in request URL")
			return
		}

		acceptValues := parseHeaderValues(h, "Accept")
		if len(acceptValues) == 0 {
			ctx, _ = log.WriteWarnWithFields(
				ctx,
				fmt.Sprintf("%s: Accept header is required", responseutils.FormatErr),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, "Accept header is required")
			return
		}

		if !hasHeaderToken(acceptValues, "application/fhir+json") {
			ctx, _ = log.WriteWarnWithFields(
				ctx,
				fmt.Sprintf("%s: application/fhir+json is the only supported response format", responseutils.FormatErr),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, "application/fhir+json is the only supported response format")
			return
		}

		preferValues := parseHeaderValues(h, "Prefer")
		if len(preferValues) == 0 {
			ctx, _ = log.WriteWarnWithFields(
				ctx,
				fmt.Sprintf("%s: Prefer header is required", responseutils.FormatErr),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, "Prefer header is required")
			return
		}

		if !hasHeaderToken(preferValues, "respond-async") {
			ctx, _ = log.WriteWarnWithFields(
				ctx,
				fmt.Sprintf("%s: Only asynchronous responses are supported", responseutils.FormatErr),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.OpOutcome(ctx, w, http.StatusBadRequest, responseutils.FormatErr, "Only asynchronous responses are supported")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// hasHeaderToken checks if targetToken exists within the parsed header values,
// stripping optional parameters (anything after ';') and performing case-insensitive matching.
func hasHeaderToken(values []string, targetToken string) bool {
	for _, v := range values {
		token, _, _ := strings.Cut(v, ";")
		if strings.EqualFold(strings.TrimSpace(token), targetToken) {
			return true
		}
	}
	return false
}

// parseHeaderValues extracts all values from a header, handling both multi-line
// header fields and comma-separated parameter lists within a single header value.
// It returns a slice of trimmed tokens.
func parseHeaderValues(h http.Header, headerName string) []string {
	values := h.Values(headerName)
	if len(values) == 0 {
		return nil
	}
	results := make([]string, 0, len(values))
	for _, rawValue := range values {
		for _, part := range strings.Split(rawValue, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				results = append(results, trimmed)
			}
		}
	}
	return results
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
	OpOutcome(context.Context, http.ResponseWriter, int, string, string)
}

// non auth resp writers should return an error on an invalid version
func GetRespWriter(path string) (fhirResponseWriter, error) {
	if strings.Contains(path, "/v1/") {
		return responseutils.NewFhirResponseWriter(), nil
	} else if strings.Contains(path, "/v2/") {
		return responseutilsv2.NewFhirResponseWriter(), nil
	} else if strings.Contains(path, fmt.Sprintf("/%s/", constants.V3Version)) {
		return responseutilsv3.NewFhirResponseWriter(), nil
	} else {
		return responseutils.NewFhirResponseWriter(), errors.New("invalid or missing version")
	}
}
