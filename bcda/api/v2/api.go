package v2

import (
	"net/http"

	"github.com/go-chi/chi"

	api "github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

const (
	groupAll = "all"
)

/*
	swagger:route GET /api/v2/Patient/$export bulkData BulkPatientRequest

	Start data export for all supported resource types

	Initiates a job to collect data from the Blue Button API for your ACO. Supported resource types are Patient, Coverage, and ExplanationOfBenefit.

	Produces:
	- application/fhir+json

	Security:
		bearer_token:

	Responses:
		202: BulkRequestResponse
		400: badRequestResponse
		401: invalidCredentials
		429: tooManyRequestsResponse
		500: errorResponse
*/
func BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	resourceTypes, err := api.ValidateRequest(r)
	if err != nil {
		responseutils.WriteError(err, w, http.StatusBadRequest)
		return
	}
	retrieveNewBeneHistData := false // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	api.BulkRequest(resourceTypes, w, r, retrieveNewBeneHistData)
}

/*
	swagger:route GET /api/v2/Group/{groupId}/$export bulkData bulkGroupRequest

    Start data export (for the specified group identifier) for all supported resource types

	Initiates a job to collect data from the Blue Button API for your ACO. The only Group identifier supported by the system is `all`.  The `all` identifier returns data for the group of all patients attributed to the requesting ACO.  If used when specifying `_since`: all claims data which has been updated since the specified date will be returned for beneficiaries which have been attributed to the ACO since before the specified date; and all historical claims data will be returned for beneficiaries which have been newly attributed to the ACO since the specified date.

	Produces:
	- application/fhir+json

	Security:
		bearer_token:

	Responses:
		202: BulkRequestResponse
		400: badRequestResponse
		401: invalidCredentials
		429: tooManyRequestsResponse
		500: errorResponse
*/
func BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	retrieveNewBeneHistData := false

	groupID := chi.URLParam(r, "groupId")
	if groupID == groupAll {
		resourceTypes, err := api.ValidateRequest(r)
		if err != nil {
			responseutils.WriteError(err, w, http.StatusBadRequest)
			return
		}

		// Set flag to retrieve new beneficiaries' historical data if _since param is provided and feature is turned on
		_, ok := r.URL.Query()["_since"]
		if ok && utils.GetEnvBool("BCDA_ENABLE_NEW_GROUP", false) {
			retrieveNewBeneHistData = true
		}

		api.BulkRequest(resourceTypes, w, r, retrieveNewBeneHistData)
	} else {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Invalid group ID")
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}
}
