package v3

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/r4"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

type ApiV3 struct {
	handler *api.Handler
	db      *sql.DB
}

func NewApiV3(db *sql.DB, pool *pgxv5Pool.Pool) *ApiV3 {
	resources, ok := service.GetClaimTypesMap([]string{
		"Patient",
		"Coverage",
		"ExplanationOfBenefit",
	}...)

	if !ok {
		panic("Failed to configure resource DataTypes")
	} else {
		h := api.NewHandler(resources, constants.BFDV3Path, constants.V3Version, db, pool)
		return &ApiV3{handler: h, db: db}
	}
}

// NOTE: The below are all just copies of v3/api

/*
swagger:route GET /api/v3/Patient/$export bulkDatav3 bulkPatientRequestv3

Start FHIR R4 data export for all supported resource types.

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
func (a ApiV3) BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	a.handler.BulkPatientRequest(w, r)
}

/*
swagger:route GET /api/v3/Group/{groupId}/$export bulkDatav3 bulkGroupRequestv3

# Start FHIR R4 data export (for the specified group identifier) for all supported resource types

Initiates a job to collect data from the Blue Button API for your ACO. The only Group identifier supported by the system are `all` and `runout`.

The `all` identifier returns data for the group of all patients attributed to the requesting ACO.  If used when specifying `_since`: all claims data which has been updated since the specified date will be returned for beneficiaries which have been attributed to the ACO since before the specified date; and all historical claims data will be returned for beneficiaries which have been newly attributed to the ACO since the specified date.

The `runout` identifier returns claims runouts data.

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
func (a ApiV3) BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	a.handler.BulkGroupRequest(w, r)
}

/*
swagger:route GET /api/v3/jobs/{jobId} jobv3 jobStatusv3

# Get job status

Returns the current status of an export job.

Produces:
- application/fhir+json

Schemes: http, https

Security:

	bearer_token:

Responses:

	202: jobStatusResponse
	200: completedJobResponse
	400: badRequestResponse
	401: invalidCredentials
	404: notFoundResponse
	410: goneResponse
	500: errorResponse
*/
func (a ApiV3) JobStatus(w http.ResponseWriter, r *http.Request) {
	a.handler.JobStatus(w, r)
}

/*
swagger:route GET /api/v3/jobs jobv3 jobsStatusv3

# Get jobs statuses

Returns the current statuses of export jobs. Supported status types are Completed, Archived, Expired, Failed, FailedExpired,
In Progress, Pending, Cancelled, and CancelledExpired. If no status(s) is provided, all jobs will be returned.

Note on job status to fhir task resource status mapping:
Due to the fhir task status field having a smaller set of values, the following statuses will be set to different fhir values in the response

Archived, Expired -> Completed
FailedExpired -> Failed
Pending -> In Progress
CancelledExpired -> Cancelled

Though the status name has been remapped the response will still only contain jobs pertaining to the provided job status in the request.

Produces:
- application/fhir+json

Schemes: http, https

Security:

	bearer_token:

Responses:

	200: jobsStatusResponse
	400: badRequestResponse
	401: invalidCredentials
	404: notFoundResponse
	410: goneResponse
	500: errorResponse
*/
func (a ApiV3) JobsStatus(w http.ResponseWriter, r *http.Request) {
	a.handler.JobsStatus(w, r)
}

/*
swagger:route DELETE /api/v3/jobs/{jobId} jobv3 deleteJobv3

# Cancel a job

Cancels a currently running job.

Produces:
- application/fhir+json

Schemes: http, https

Security:

	bearer_token:

Responses:

	202: deleteJobResponse
	400: badRequestResponse
	401: invalidCredentials
	404: notFoundResponse
	410: goneResponse
	500: errorResponse
*/
func (a ApiV3) DeleteJob(w http.ResponseWriter, r *http.Request) {
	a.handler.DeleteJob(w, r)
}

/*
swagger:route GET /api/v3/attribution_status attributionStatusv3 attributionStatusv3

# Get attribution status

Returns the status of the latest ingestion for attribution and claims runout files. The response will contain the Type to identify which ingestion and a Timestamp for the last time it was updated.

Produces:
- application/json

Schemes: http, https

Security:

	bearer_token:

Responses:

	200: AttributionFileStatusResponse
	404: notFoundResponse
*/
func (a ApiV3) AttributionStatus(w http.ResponseWriter, r *http.Request) {
	a.handler.AttributionStatus(w, r)
}

/*
swagger:route GET /api/v3/metadata metadatav3 metadata

# Get metadata

Returns metadata about the API.

Produces:
- application/fhir+json

Schemes: http, https

Responses:

	200: MetadataResponse
*/
func (a ApiV3) Metadata(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	bbServer := conf.GetEnv("BB_SERVER_LOCATION")

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	statement := &r4.CapabilityStatement{
		ResourceType: "CapabilityStatement",
		Status:       r4.PublicationStatusActive,
		Date:         dt.UTC().Format("2006-01-02T15:04:05Z"),
		Publisher:    constants.PublisherName,
		Kind:         r4.CapabilityStatementKindInstance,
		Instantiates: []string{
			bbServer + "/v3/fhir/metadata",
			"http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data",
		},
		Software: r4.Software{
			Name:        constants.SoftwareName,
			Version:     constants.Version,
			ReleaseDate: dt.UTC().Format("2006-01-02T15:04:05Z"),
		},
		Implementation: r4.Implementation{
			Description: constants.SoftwareDescription,
			Url:         baseURL,
		},
		FhirVersion: "4.0.1",
		Format: []string{
			constants.JsonContentType,
			constants.FHIRJsonContentType,
		},
		Rest: []r4.CapabilityStatementRest{
			{
				Mode: r4.RestfulCapabilityModeServer,
				Security: &r4.Security{
					Cors: true,
					Service: []r4.CodeableConcept{
						{
							Coding: []r4.Coding{
								{
									Display: "OAuth",
									Code:    "OAuth",
									System:  constants.RestfulSecurityServiceSystem,
								},
							},
							Text: "OAuth",
						},
					},
					Extension: []r4.Extension{
						{
							Url: constants.SmartOAuthURIsExtensionURL,
							Extension: []r4.Extension{
								{
									Url:      "token",
									ValueUri: baseURL + "/auth/token",
								},
							},
						},
					},
				},
				Interaction: []r4.Interaction{
					{
						Code: r4.SystemRestfulInteractionBatch,
					},
					{
						Code: r4.SystemRestfulInteractionSearchSystem,
					},
				},
				Resource: []r4.RestResource{
					{
						Type: r4.ResourceTypeCodePatient,
						Operation: []r4.RestOperation{
							{
								Name:          "export",
								Definition:    "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export",
								Documentation: "By default, the patient $export will return ExplanationOfBenefit resources with a meta.tag with a system of 'https://bluebutton.cms.gov/fhir/CodeSystem/System-Type' and code of either NationalClaimsHistory or DDPS. In order to return ExplanationOfBenefit resources with other system types (like SharedSystem), use the _typeFilter parameter.",
							},
						},
						SearchParam: []r4.SearchParam{
							restResourceSearchParam("_since", "date", "Return resources updated after the date provided for existing and newly attributed enrollees."),
							restResourceSearchParam("_type", "string", "Comma-delimited list of FHIR resource types to include in the export. By default, all supported resource types are returned."),
							restResourceSearchParam("_typeFilter", "string", "Use a URL-encoded FHIR subquery to further-refine patient export results."),
						},
					},
					{
						Type: r4.ResourceTypeCodeGroup,
						Operation: []r4.RestOperation{
							{
								Name:          "export",
								Definition:    "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export",
								Documentation: "By default, the group $export will return ExplanationOfBenefit resources with a meta.tag with a system of 'https://bluebutton.cms.gov/fhir/CodeSystem/System-Type' and code of either NationalClaimsHistory or DDPS. In order to return ExplanationOfBenefit resources with other system types (like SharedSystem), use the _typeFilter parameter.",
							},
						},
						SearchParam: []r4.SearchParam{
							restResourceSearchParam("_since", "date", "Return resources updated after the date provided for existing enrollees and all resources for newly attributed enrollees."),
							restResourceSearchParam("_type", "string", "Comma-delimited list of FHIR resource types to include in the export. By default, all supported resource types are returned."),
							restResourceSearchParam("_typeFilter", "string", "Use a URL-encoded FHIR subquery to further-refine group export results."),
						},
					},
					{
						Type: r4.ResourceTypeCodeExplanationOfBenefit,
						SearchParam: []r4.SearchParam{
							restResourceSearchParam("_tag", "token", "Filter ExplanationOfBenefit by the meta.tag element. Pass full token as <system>|<code>. Supported codes in the 'https://bluebutton.cms.gov/fhir/CodeSystem/System-Type' system are: 'SharedSystem', 'NationalClaimsHistory', and 'DDPS'. By Default, only NationalClaimsHistory and DDPS claims will be returned."),
							restResourceSearchParam("outcome", "token", "Filter ExplanationOfBenefit by the outcome element. Supported values: 'partial' and 'complete'."),
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(statement)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Errorf("Failed to marshal Capability Statement %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	if _, err = w.Write(b); err != nil { // #nosec G705
		log.API.WithField("resp_status", http.StatusInternalServerError).Errorf("Failed to write data %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func restResourceSearchParam(n string, t string, d string) r4.SearchParam {
	p := r4.SearchParam{
		Name:          n,
		Type:          t,
		Documentation: d,
	}

	return p
}
