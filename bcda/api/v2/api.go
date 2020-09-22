package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"

	api "github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
	log "github.com/sirupsen/logrus"
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

/*
	swagger:route GET /api/v2/metadata metadata metadata

	Get metadata

	Returns metadata about the API.

	Produces:
	- application/fhir+json

	Schemes: http, https

	Responses:
		200: MetadataResponse
*/
func Metadata(w http.ResponseWriter, r *http.Request) {
	const dateFormat = "2006-01-02"

	dt := time.Now()
	useCors := true
	bbServer := os.Getenv("BB_SERVER_LOCATION")

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)
	statement := fhir.CapabilityStatement{
		Status:    fhir.PublicationStatusActive,
		Date:      dt.Format(dateFormat),
		Publisher: getStringPtr("Centers for Medicare & Medicaid Services"),
		Kind:      fhir.CapabilityStatementKindInstance,
		// TODO (BCDA-3732): Update to r4 once endpoint is available
		Instantiates: []string{bbServer + "/baseDstu3/metadata/", "http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data"},
		Software: &fhir.CapabilityStatementSoftware{
			Name:        "Beneficiary Claims Data API",
			Version:     &constants.Version,
			ReleaseDate: getStringPtr(dt.Format(dateFormat)),
		},
		Implementation: &fhir.CapabilityStatementImplementation{
			Description: "The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their prospectively assigned or assignable beneficiaries.",
			Url:         &baseURL,
		},
		FhirVersion: fhir.FHIRVersion4_0_1,
		Format:      []string{"application/json", "application/fhir+json"},
		Rest: []fhir.CapabilityStatementRest{
			{
				Mode: fhir.RestfulCapabilityModeServer,
				Security: &fhir.CapabilityStatementRestSecurity{
					Cors: &useCors,
					Service: []fhir.CodeableConcept{
						{
							Coding: []fhir.Coding{
								{
									Display: getStringPtr("OAuth"),
									Code:    getStringPtr("OAuth"),
									System:  getStringPtr("http://terminology.hl7.org/CodeSystem/restful-security-service"),
								},
							},
							Text: getStringPtr("OAuth"),
						},
					},
				},
				Interaction: []fhir.CapabilityStatementRestInteraction{
					{
						Code: fhir.SystemRestfulInteractionBatch,
					},
					{
						Code: fhir.SystemRestfulInteractionSearchSystem,
					},
				},
				Resource: []fhir.CapabilityStatementRestResource{
					{
						Type: fhir.ResourceTypePatient,
						Operation: []fhir.CapabilityStatementRestResourceOperation{
							{
								Name:       "patient-export",
								Definition: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export",
							},
						},
					},
					{
						Type: fhir.ResourceTypeGroup,
						Operation: []fhir.CapabilityStatementRestResourceOperation{
							{
								Name:       "group-export",
								Definition: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export",
							},
						},
					},
				},
			},
		},
	}

	b, err := statement.MarshalJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Need this logic since extensions currently do not support polymorphic data types
	// See: https://github.com/samply/golang-fhir-models/issues/1
	extension := []map[string]interface{}{
		{"url": "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris",
			"extension": []map[string]interface{}{
				{"url": "token", "valueUri": baseURL},
			},
		},
	}
	var obj map[string]interface{}
	if err = json.Unmarshal(b, &obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// The field we're trying to update will always be found at rest[0].security.extension
	obj["rest"].([]interface{})[0].(map[string]interface{})["security"].(map[string]interface{})["extension"] = extension
	if b, err = json.Marshal(obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(b); err != nil {
		log.Errorf("Failed to write data %s", err.Error())
	}
}

func getStringPtr(value string) *string {
	return &value
}
