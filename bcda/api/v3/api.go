package v3

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	fhirresources "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhircapabilitystatement "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
	fhirvaluesets "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

type ApiV3 struct {
	handler    *api.Handler
	marshaller *jsonformat.Marshaller
	db         *sql.DB
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
		// Ensure that we write the serialized FHIR resources as a single line.
		// Needed to comply with the NDJSON format that we are using.
		marshaller, err := jsonformat.NewMarshaller(false, "", "", fhirversion.R4)
		if err != nil {
			log.API.Fatalf("Failed to create marshaller %s", err)
		}
		return &ApiV3{marshaller: marshaller, handler: h, db: db}
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

	    Start FHIR R4 data export (for the specified group identifier) for all supported resource types

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
swagger:route GET /api/v3/attribution_status attributionStatusv3 attributionStatus

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

	statement := &fhircapabilitystatement.CapabilityStatement{
		Status: &fhircapabilitystatement.CapabilityStatement_StatusCode{Value: fhircodes.PublicationStatusCode_ACTIVE},
		Date: &fhirdatatypes.DateTime{
			ValueUs:   dt.UTC().UnixNano() / int64(time.Microsecond),
			Timezone:  time.UTC.String(),
			Precision: fhirdatatypes.DateTime_SECOND,
		},
		Publisher: &fhirdatatypes.String{Value: "Centers for Medicare & Medicaid Services"},
		Kind:      &fhircapabilitystatement.CapabilityStatement_KindCode{Value: fhircodes.CapabilityStatementKindCode_INSTANCE},
		Instantiates: []*fhirdatatypes.Canonical{
			{Value: bbServer + constants.BFDV3Path + "/metadata"},
			{Value: "http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data"},
		},
		Software: &fhircapabilitystatement.CapabilityStatement_Software{
			Name:    &fhirdatatypes.String{Value: "Beneficiary Claims Data API"},
			Version: &fhirdatatypes.String{Value: constants.Version},
			ReleaseDate: &fhirdatatypes.DateTime{
				ValueUs:   dt.UTC().UnixNano() / int64(time.Microsecond),
				Timezone:  time.UTC.String(),
				Precision: fhirdatatypes.DateTime_SECOND,
			},
		},
		Implementation: &fhircapabilitystatement.CapabilityStatement_Implementation{
			Description: &fhirdatatypes.String{Value: "The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their prospectively assigned or assignable beneficiaries."},
			Url:         &fhirdatatypes.Url{Value: baseURL},
		},
		FhirVersion: &fhircapabilitystatement.CapabilityStatement_FhirVersionCode{Value: fhircodes.FHIRVersionCode_V_4_0_1},
		Format: []*fhircapabilitystatement.CapabilityStatement_FormatCode{
			{Value: "application/json"},
			{Value: "application/fhir+json"},
		},
		Rest: []*fhircapabilitystatement.CapabilityStatement_Rest{
			{
				Mode: &fhircapabilitystatement.CapabilityStatement_Rest_ModeCode{Value: fhircodes.RestfulCapabilityModeCode_SERVER},
				Security: &fhircapabilitystatement.CapabilityStatement_Rest_Security{
					Cors: &fhirdatatypes.Boolean{Value: true},
					Service: []*fhirdatatypes.CodeableConcept{
						{
							Coding: []*fhirdatatypes.Coding{
								{
									Display: &fhirdatatypes.String{Value: "OAuth"},
									Code:    &fhirdatatypes.Code{Value: "OAuth"},
									System:  &fhirdatatypes.Uri{Value: "http://terminology.hl7.org/CodeSystem/restful-security-service"},
								},
							},
							Text: &fhirdatatypes.String{Value: "OAuth"},
						},
					},
					Extension: []*fhirdatatypes.Extension{
						{
							Url: &fhirdatatypes.Uri{Value: "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris"},
							Extension: []*fhirdatatypes.Extension{
								{
									Url: &fhirdatatypes.Uri{Value: "token"},
									Value: &fhirdatatypes.Extension_ValueX{
										Choice: &fhirdatatypes.Extension_ValueX_Uri{
											Uri: &fhirdatatypes.Uri{Value: baseURL + "/auth/token"},
										},
									},
								},
							},
						},
					},
				},
				Interaction: []*fhircapabilitystatement.CapabilityStatement_Rest_SystemInteraction{
					{
						Code: &fhircapabilitystatement.CapabilityStatement_Rest_SystemInteraction_CodeType{Value: fhirvaluesets.SystemRestfulInteractionValueSet_BATCH},
					},
					{
						Code: &fhircapabilitystatement.CapabilityStatement_Rest_SystemInteraction_CodeType{Value: fhirvaluesets.SystemRestfulInteractionValueSet_SEARCH_SYSTEM},
					},
				},
				Resource: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource{
					{
						Type: &fhircapabilitystatement.CapabilityStatement_Rest_Resource_TypeCode{Value: fhircodes.ResourceTypeCode_PATIENT},
						Operation: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource_Operation{
							{
								Name:       &fhirdatatypes.String{Value: "export"},
								Definition: &fhirdatatypes.Canonical{Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
							},
						},
						SearchParam: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam{
							{
								Name:          &fhirdatatypes.String{Value: "_since"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_DATE},
								Documentation: &fhirdatatypes.Markdown{Value: "Return resources updated after the date provided for existing and newly attributed enrollees."},
							},
							{
								Name:          &fhirdatatypes.String{Value: "_type"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_STRING},
								Documentation: &fhirdatatypes.Markdown{Value: "Comma-delimited list of FHIR resource types to include in the export. By default, all supported resource types are returned."},
							},
							{
								Name:          &fhirdatatypes.String{Value: "_typeFilter"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_STRING},
								Documentation: &fhirdatatypes.Markdown{Value: "Use a URL-encoded FHIR subquery to further-refine export results."},
							},
						},
					},
					{
						Type: &fhircapabilitystatement.CapabilityStatement_Rest_Resource_TypeCode{Value: fhircodes.ResourceTypeCode_GROUP},
						Operation: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource_Operation{
							{
								Name:       &fhirdatatypes.String{Value: "export"},
								Definition: &fhirdatatypes.Canonical{Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
							},
						},
						SearchParam: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam{
							{
								Name:          &fhirdatatypes.String{Value: "_since"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_DATE},
								Documentation: &fhirdatatypes.Markdown{Value: "Return resources updated after the date provided for existing enrollees and all resources for newly attributed enrollees."},
							},
							{
								Name:          &fhirdatatypes.String{Value: "_type"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_STRING},
								Documentation: &fhirdatatypes.Markdown{Value: "Comma-delimited list of FHIR resource types to include in the export. By default, all supported resource types are returned."},
							},
							{
								Name:          &fhirdatatypes.String{Value: "_typeFilter"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_STRING},
								Documentation: &fhirdatatypes.Markdown{Value: "Use a URL-encoded FHIR subquery to further-refine export results."},
							},
						},
					},
					{
						Type: &fhircapabilitystatement.CapabilityStatement_Rest_Resource_TypeCode{Value: fhircodes.ResourceTypeCode_EXPLANATION_OF_BENEFIT},
						SearchParam: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam{
							{
								Name:          &fhirdatatypes.String{Value: "_tag"},
								Type:          &fhircapabilitystatement.CapabilityStatement_Rest_Resource_SearchParam_TypeCode{Value: fhircodes.SearchParamTypeCode_TOKEN},
								Documentation: &fhirdatatypes.Markdown{Value: "Filter claims by adjudication status: either Adjudicated or PartiallyAdjudicated"},
							},
						},
					},
				},
			},
		},
	}

	resource := &fhirresources.ContainedResource{
		OneofResource: &fhirresources.ContainedResource_CapabilityStatement{CapabilityStatement: statement},
	}
	b, err := a.marshaller.Marshal(resource)
	if err != nil {
		log.API.Errorf("Failed to marshal Capability Statement %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	if _, err = w.Write(b); err != nil {
		log.API.Errorf("Failed to write data %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
