package responseutils

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

var marshaller *jsonformat.Marshaller

func init() {
	var err error

	// Ensure that we write the serialized FHIR resources as a single line.
	// Needed to comply with the NDJSON format that we are using.
	marshaller, err = jsonformat.NewMarshaller(false, "", "", jsonformat.STU3)
	if err != nil {
		log.Fatalf("Failed to create marshaller %s", err)
	}
}

type ResponseWriter struct{}

func NewResponseWriter() ResponseWriter {
	return ResponseWriter{}
}

func (r ResponseWriter) Exception(w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, errType, errMsg)
	WriteError(oo, w, statusCode)
}

func (r ResponseWriter) NotFound(w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_NOT_FOUND, errType, errMsg)
	WriteError(oo, w, statusCode)
}

func CreateOpOutcome(severity fhircodes.IssueSeverityCode_Value, code fhircodes.IssueTypeCode_Value,
	detailsCode, detailsDisplay string) *fhirmodels.OperationOutcome {

	return &fhirmodels.OperationOutcome{
		Issue: []*fhirmodels.OperationOutcome_Issue{
			{
				Severity: &fhircodes.IssueSeverityCode{Value: severity},
				Code:     &fhircodes.IssueTypeCode{Value: code},
				Details: &fhirdatatypes.CodeableConcept{
					Coding: []*fhirdatatypes.Coding{
						{
							Code: &fhirdatatypes.Code{Value: detailsCode},
							System: &fhirdatatypes.Uri{
								Value: "http://hl7.org/fhir/ValueSet/operation-outcome",
							},
							Display: &fhirdatatypes.String{Value: detailsDisplay},
						},
					},
					Text: &fhirdatatypes.String{Value: detailsDisplay},
				},
			},
		},
	}
}

func WriteError(outcome *fhirmodels.OperationOutcome, w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := WriteOperationOutcome(w, outcome)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func WriteOperationOutcome(w io.Writer, outcome *fhirmodels.OperationOutcome) (int, error) {
	resource := &fhirmodels.ContainedResource{
		OneofResource: &fhirmodels.ContainedResource_OperationOutcome{OperationOutcome: outcome},
	}
	outcomeJSON, err := marshaller.Marshal(resource)
	if err != nil {
		return -1, err
	}

	return w.Write(outcomeJSON)
}

func CreateCapabilityStatement(reldate time.Time, relversion, baseurl string) *fhirmodels.CapabilityStatement {
	bbServer := conf.GetEnv("BB_SERVER_LOCATION")
	statement := &fhirmodels.CapabilityStatement{
		Status: &fhircodes.PublicationStatusCode{Value: fhircodes.PublicationStatusCode_ACTIVE},
		Date: &fhirdatatypes.DateTime{
			ValueUs:   reldate.UTC().UnixNano() / int64(time.Microsecond),
			Timezone:  time.UTC.String(),
			Precision: fhirdatatypes.DateTime_SECOND,
		},
		Publisher: &fhirdatatypes.String{Value: "Centers for Medicare & Medicaid Services"},
		Kind:      &fhircodes.CapabilityStatementKindCode{Value: fhircodes.CapabilityStatementKindCode_INSTANCE},
		Instantiates: []*fhirdatatypes.Uri{
			{Value: bbServer + "/baseDstu3/metadata/"},
			{Value: "http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data"},
		},
		Software: &fhirmodels.CapabilityStatement_Software{
			Name:    &fhirdatatypes.String{Value: "Beneficiary Claims Data API"},
			Version: &fhirdatatypes.String{Value: relversion},
			ReleaseDate: &fhirdatatypes.DateTime{
				ValueUs:   reldate.UTC().UnixNano() / int64(time.Microsecond),
				Timezone:  time.UTC.String(),
				Precision: fhirdatatypes.DateTime_SECOND,
			},
		},
		Implementation: &fhirmodels.CapabilityStatement_Implementation{
			Description: &fhirdatatypes.String{Value: "The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their prospectively assigned or assignable beneficiaries."},
			Url:         &fhirdatatypes.Uri{Value: baseurl},
		},
		FhirVersion:   &fhirdatatypes.Id{Value: "3.0.1"},
		AcceptUnknown: &fhircodes.UnknownContentCodeCode{Value: fhircodes.UnknownContentCodeCode_EXTENSIONS},
		Format: []*fhirdatatypes.MimeTypeCode{
			{Value: "application/json"},
			{Value: "application/fhir+json"},
		},
		Rest: []*fhirmodels.CapabilityStatement_Rest{
			{
				Mode: &fhircodes.RestfulCapabilityModeCode{Value: fhircodes.RestfulCapabilityModeCode_SERVER},
				Security: &fhirmodels.CapabilityStatement_Rest_Security{
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
											Uri: &fhirdatatypes.Uri{Value: baseurl + "/auth/token"},
										},
									},
								},
							},
						},
					},
				},
				Interaction: []*fhirmodels.CapabilityStatement_Rest_SystemInteraction{
					{
						Code: &fhircodes.SystemRestfulInteractionCode{Value: fhircodes.SystemRestfulInteractionCode_BATCH},
					},
					{
						Code: &fhircodes.SystemRestfulInteractionCode{Value: fhircodes.SystemRestfulInteractionCode_SEARCH_SYSTEM},
					},
				},
				Operation: []*fhirmodels.CapabilityStatement_Rest_Operation{
					{
						Name: &fhirdatatypes.String{Value: "patient-export"},
						Definition: &fhirdatatypes.Reference{
							Reference: &fhirdatatypes.Reference_Uri{
								Uri: &fhirdatatypes.String{Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
							},
						},
					},
					{
						Name: &fhirdatatypes.String{Value: "group-export"},
						Definition: &fhirdatatypes.Reference{
							Reference: &fhirdatatypes.Reference_Uri{
								Uri: &fhirdatatypes.String{Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
							},
						},
					},
				},
			},
		},
	}
	return statement
}
func WriteCapabilityStatement(statement *fhirmodels.CapabilityStatement, w http.ResponseWriter) {
	resource := &fhirmodels.ContainedResource{
		OneofResource: &fhirmodels.ContainedResource_CapabilityStatement{CapabilityStatement: statement},
	}
	statementJSON, err := marshaller.Marshal(resource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(statementJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
