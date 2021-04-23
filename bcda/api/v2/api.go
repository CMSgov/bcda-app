package v2

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	fhirresources "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhircapabilitystatement "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
	fhirvaluesets "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"

	api "github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

var (
	h          *api.Handler
	marshaller *jsonformat.Marshaller
)

func init() {
	var err error

	h = api.NewHandler([]string{"Patient", "Coverage"}, "/v2/fhir")
	// Ensure that we write the serialized FHIR resources as a single line.
	// Needed to comply with the NDJSON format that we are using.
	marshaller, err = jsonformat.NewMarshaller(false, "", "", jsonformat.R4)
	if err != nil {
		log.API.Fatalf("Failed to create marshaller %s", err)
	}
}

/*
	swagger:route GET /api/v2/Patient/$export bulkDataV2 bulkPatientRequestV2

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
func BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	h.BulkPatientRequest(w, r)
}

/*
	swagger:route GET /api/v2/Group/{groupId}/$export bulkDataV2 bulkGroupRequestV2

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
func BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	h.BulkGroupRequest(w, r)
}

/*
	swagger:route GET /api/v2/metadata metadataV2 metadata

	Get metadata

	Returns metadata about the API.

	Produces:
	- application/fhir+json

	Schemes: http, https

	Responses:
		200: MetadataResponse
*/
func Metadata(w http.ResponseWriter, r *http.Request) {
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
			{Value: bbServer + "/v2/fhir/metadata"},
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
								Name:       &fhirdatatypes.String{Value: "patient-export"},
								Definition: &fhirdatatypes.Canonical{Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
							},
						},
					},
					{
						Type: &fhircapabilitystatement.CapabilityStatement_Rest_Resource_TypeCode{Value: fhircodes.ResourceTypeCode_GROUP},
						Operation: []*fhircapabilitystatement.CapabilityStatement_Rest_Resource_Operation{
							{
								Name:       &fhirdatatypes.String{Value: "group-export"},
								Definition: &fhirdatatypes.Canonical{Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
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
	b, err := marshaller.Marshal(resource)
	if err != nil {
		log.API.Errorf("Failed to marshal Capability Statement %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(b); err != nil {
		log.API.Errorf("Failed to write data %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
