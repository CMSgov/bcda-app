package middleware

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/responseutils"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
)

var supportedOutputFormats = map[string]struct{}{
	"ndjson":                  {},
	"application/fhir+ndjson": {},
	"application/ndjson":      {}}

type RequestParameters struct {
	Since         time.Time
	ResourceTypes []string
	Version       string // e.g. v1, v2
}

// requestkey is an unexported context key to avoid collisions
type requestkey int

var rk requestkey

func NewRequestParametersContext(ctx context.Context, rp RequestParameters) context.Context {
	return context.WithValue(ctx, rk, rp)
}

func RequestParametersFromContext(ctx context.Context) (RequestParameters, bool) {
	rp, ok := ctx.Value(rk).(RequestParameters)
	return rp, ok
}

func ValidateRequestURL(next Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rp RequestParameters

		//validate "_outputFormat" parameter
		params, ok := r.URL.Query()["_outputFormat"]
		if ok {
			if _, found := supportedOutputFormats[params[0]]; !found {
				errMsg := fmt.Sprintf("_outputFormat parameter must be one of %v", getKeys(supportedOutputFormats))
				oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.FormatErr, errMsg)
				responseutils.WriteError(oo, w, http.StatusBadRequest)
				return
			}
		}

		// we do not support "_elements" parameter
		_, ok = r.URL.Query()["_elements"]
		if ok {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "Invalid parameter: this server does not support the _elements parameter.")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		// Check and see if the user has a duplicated the query parameter symbol (?)
		// e.g. /api/v1/Patient/$export?_type=ExplanationOfBenefit&?_since=2020-09-13T08:00:00.000-05:00
		for key := range r.URL.Query() {
			if strings.HasPrefix(key, "?") {
				oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.FormatErr, "Invalid parameter: query parameters cannot start with ?")
				responseutils.WriteError(oo, w, http.StatusBadRequest)
				return
			}
		}

		// validate optional "_since" parameter
		params, ok = r.URL.Query()["_since"]
		if ok {
			sinceDate, err := time.Parse(time.RFC3339Nano, params[0])
			if err != nil {
				oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.FormatErr, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format.")
				responseutils.WriteError(oo, w, http.StatusBadRequest)
				return
			} else if sinceDate.After(time.Now()) {
				oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.FormatErr, "Invalid date format supplied in _since parameter. Date must be a date that has already passed")
				responseutils.WriteError(oo, w, http.StatusBadRequest)
			}
			rp.Since = sinceDate
		}

		// validate no duplicate resource types
		var resourceTypes []string
		params, ok = r.URL.Query()["_type"]
		if ok {
			resourceMap := make(map[string]struct{})
			resourceTypes = strings.Split(params[0], ",")
			for _, resource := range resourceTypes {
				if _, ok := resourceMap[resource]; !ok {
					resourceMap[resource] = struct{}{}
				} else {
					errMsg := fmt.Sprintf("Repeated resource type %s", resource)
					oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, errMsg)
					responseutils.WriteError(oo, w, http.StatusBadRequest)
					return
				}
			}
			rp.ResourceTypes = resourceTypes
		}

		// Get API version
		version, err := getVersion(r.URL.Path)
		if err != nil {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.FormatErr, err.Error())
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}
		rp.Version = version

		r = r.WithContext(NewRequestParametersContext(r.Context(), rp))

		next(w, r, rp)
	})
}

func ValidateRequestHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header

		acceptHeader := h.Get("Accept")
		preferHeader := h.Get("Prefer")

		if acceptHeader == "" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "Accept header is required")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		} else if acceptHeader != "application/fhir+json" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "application/fhir+json is the only supported response format")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		if preferHeader == "" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "Prefer header is required")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		} else if preferHeader != "respond-async" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "Only asynchronous responses are supported")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
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

var versionExp = regexp.MustCompile(`\/api\/(.*)\/[Patient|Group].*`)

func getVersion(path string) (string, error) {
	parts := versionExp.FindStringSubmatch(path)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected path provided %s", path)
	}
	return parts[1], nil
}
