package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/responseutils"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
)

var supportedOutputFormats = map[string]struct{}{
	"ndjson":                  struct{}{},
	"application/fhir+ndjson": struct{}{},
	"application/ndjson":      struct{}{}}

func ValidateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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
		}

		// validate optional "_type" parameter
		var resourceTypes []string
		params, ok = r.URL.Query()["_type"]
		if ok {
			resourceMap := make(map[string]struct{})
			resourceTypes = strings.Split(params[0], ",")
			for _, resource := range resourceTypes {
				if _, ok := resourceMap[resource]; !ok {
					resourceMap[resource] = struct{}{}
				} else {
					errMsg := fmt.Sprintf("Repeated resource type %s")
					oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr,"Repeated resource type")
					return nil, oo
				}
			}
		}

	})
}

func getKeys(kv map[string]struct{}) []string {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	return keys
}
