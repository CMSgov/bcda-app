package responseutils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/google/fhir/go/jsonformat"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	fhirmodelCR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhirmodelCS "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
	fhirmodelOO "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/operation_outcome_go_proto"
	fhirmodelT "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/task_go_proto"
	fhirvaluesets "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"
)

var marshaller *jsonformat.Marshaller

func init() {
	var err error

	// Ensure that we write the serialized FHIR resources as a single line.
	// Needed to comply with the NDJSON format that we are using.
	marshaller, err = jsonformat.NewMarshaller(false, "", "", jsonformat.R4)
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

func (r ResponseWriter) JobsBundle(w http.ResponseWriter, jobs []*models.Job, host string) {
	jb := CreateJobsBundle(jobs, host)
	WriteBundleResponse(jb, w)
}

func CreateJobsBundle(jobs []*models.Job, host string) *fhirmodelCR.Bundle {
	var entries []*fhirmodelCR.Bundle_Entry

	// generate bundle task entries
	for _, job := range jobs {
		entry := CreateJobsBundleEntry(job, host)
		entries = append(entries, entry)
	}

	return &fhirmodelCR.Bundle{
		Type:  &fhirmodelCR.Bundle_TypeCode{Value: fhircodes.BundleTypeCode_SEARCHSET},
		Total: &fhirdatatypes.UnsignedInt{Value: uint32(len(jobs))},
		Entry: entries,
	}
}

func CreateJobsBundleEntry(job *models.Job, host string) *fhirmodelCR.Bundle_Entry {
	fhirStatusCode := GetFhirStatusCode(job.Status)

	return &fhirmodelCR.Bundle_Entry{
		Resource: &fhirmodelCR.ContainedResource{
			OneofResource: &fhirmodelCR.ContainedResource_Task{
				Task: &fhirmodelT.Task{
					Identifier: []*fhirdatatypes.Identifier{
						{
							Use:    &fhirdatatypes.Identifier_UseCode{Value: fhircodes.IdentifierUseCode_OFFICIAL},
							System: &fhirdatatypes.Uri{Value: host + "/api/v1/jobs"},
							Value:  &fhirdatatypes.String{Value: fmt.Sprint(job.ID)},
						},
					},
					Status: &fhirmodelT.Task_StatusCode{Value: fhirStatusCode},
					Intent: &fhirmodelT.Task_IntentCode{Value: fhirvaluesets.TaskIntentValueSet_ORDER},
					Input: []*fhirmodelT.Task_Parameter{
						{
							Type:  &fhirdatatypes.CodeableConcept{Text: &fhirdatatypes.String{Value: "BULK FHIR Export"}},
							Value: &fhirmodelT.Task_Parameter_ValueX{Choice: &fhirmodelT.Task_Parameter_ValueX_StringValue{StringValue: &fhirdatatypes.String{Value: "GET " + job.RequestURL}}},
						},
					},
					ExecutionPeriod: &fhirdatatypes.Period{
						Start: &fhirdatatypes.DateTime{
							ValueUs:   job.CreatedAt.UTC().UnixNano() / int64(time.Microsecond),
							Timezone:  time.UTC.String(),
							Precision: fhirdatatypes.DateTime_SECOND,
						},
						End: &fhirdatatypes.DateTime{
							ValueUs:   job.UpdatedAt.UTC().UnixNano() / int64(time.Microsecond),
							Timezone:  time.UTC.String(),
							Precision: fhirdatatypes.DateTime_SECOND,
						},
					},
				},
			},
		},
	}
}

func GetFhirStatusCode(status models.JobStatus) fhircodes.TaskStatusCode_Value {
	var fhirStatus fhircodes.TaskStatusCode_Value

	switch status {

	case models.JobStatusFailed, models.JobStatusFailedExpired:
		fhirStatus = fhircodes.TaskStatusCode_FAILED
	case models.JobStatusPending, models.JobStatusInProgress:
		fhirStatus = fhircodes.TaskStatusCode_IN_PROGRESS
	case models.JobStatusCompleted:
		fhirStatus = fhircodes.TaskStatusCode_COMPLETED
	case models.JobStatusArchived, models.JobStatusExpired:
		fhirStatus = fhircodes.TaskStatusCode_COMPLETED // fhir task status does not have an equivalent to `expired` or `archived`
	case models.JobStatusCancelled, models.JobStatusCancelledExpired:
		fhirStatus = fhircodes.TaskStatusCode_CANCELLED
	}

	return fhirStatus
}

func CreateOpOutcome(severity fhircodes.IssueSeverityCode_Value, code fhircodes.IssueTypeCode_Value,
	detailsCode, detailsDisplay string) *fhirmodelOO.OperationOutcome {

	return &fhirmodelOO.OperationOutcome{
		Issue: []*fhirmodelOO.OperationOutcome_Issue{
			{
				Severity: &fhirmodelOO.OperationOutcome_Issue_SeverityCode{Value: severity},
				Code:     &fhirmodelOO.OperationOutcome_Issue_CodeType{Value: code},
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

func WriteError(outcome *fhirmodelOO.OperationOutcome, w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := WriteOperationOutcome(w, outcome)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func WriteOperationOutcome(w io.Writer, outcome *fhirmodelOO.OperationOutcome) (int, error) {
	resource := &fhirmodelCR.ContainedResource{
		OneofResource: &fhirmodelCR.ContainedResource_OperationOutcome{OperationOutcome: outcome},
	}
	outcomeJSON, err := marshaller.Marshal(resource)
	if err != nil {
		return -1, err
	}

	return w.Write(outcomeJSON)
}

func CreateCapabilityStatement(reldate time.Time, relversion, baseurl string) *fhirmodelCS.CapabilityStatement {
	bbServer := conf.GetEnv("BB_SERVER_LOCATION")
	statement := &fhirmodelCS.CapabilityStatement{
		Status: &fhirmodelCS.CapabilityStatement_StatusCode{Value: fhircodes.PublicationStatusCode_ACTIVE},
		Date: &fhirdatatypes.DateTime{
			ValueUs:   reldate.UTC().UnixNano() / int64(time.Microsecond),
			Timezone:  time.UTC.String(),
			Precision: fhirdatatypes.DateTime_SECOND,
		},
		Publisher: &fhirdatatypes.String{Value: "Centers for Medicare & Medicaid Services"},
		Kind:      &fhirmodelCS.CapabilityStatement_KindCode{Value: fhircodes.CapabilityStatementKindCode_INSTANCE},
		Instantiates: []*fhirdatatypes.Canonical{
			{Value: bbServer + "/baseDstu3/metadata/"},
			{Value: "http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data"},
		},
		Software: &fhirmodelCS.CapabilityStatement_Software{
			Name:    &fhirdatatypes.String{Value: "Beneficiary Claims Data API"},
			Version: &fhirdatatypes.String{Value: relversion},
			ReleaseDate: &fhirdatatypes.DateTime{
				ValueUs:   reldate.UTC().UnixNano() / int64(time.Microsecond),
				Timezone:  time.UTC.String(),
				Precision: fhirdatatypes.DateTime_SECOND,
			},
		},
		Implementation: &fhirmodelCS.CapabilityStatement_Implementation{
			Description: &fhirdatatypes.String{Value: "The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their prospectively assigned or assignable beneficiaries."},
			Url:         &fhirdatatypes.Url{Value: baseurl},
		},
		FhirVersion: &fhirmodelCS.CapabilityStatement_FhirVersionCode{Value: fhircodes.FHIRVersionCode_V_3_0_1},
		Format: []*fhirmodelCS.CapabilityStatement_FormatCode{
			{Value: "application/json"},
			{Value: "application/fhir+json"},
		},
		Rest: []*fhirmodelCS.CapabilityStatement_Rest{
			{
				Mode: &fhirmodelCS.CapabilityStatement_Rest_ModeCode{Value: fhircodes.RestfulCapabilityModeCode_SERVER},
				Security: &fhirmodelCS.CapabilityStatement_Rest_Security{
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
				Interaction: []*fhirmodelCS.CapabilityStatement_Rest_SystemInteraction{
					{
						Code: &fhirmodelCS.CapabilityStatement_Rest_SystemInteraction_CodeType{Value: fhirvaluesets.SystemRestfulInteractionValueSet_BATCH},
					},
					{
						Code: &fhirmodelCS.CapabilityStatement_Rest_SystemInteraction_CodeType{Value: fhirvaluesets.SystemRestfulInteractionValueSet_SEARCH_SYSTEM},
					},
				},
				Operation: []*fhirmodelCS.CapabilityStatement_Rest_Resource_Operation{
					{
						Name: &fhirdatatypes.String{Value: "patient-export"},
						Definition: &fhirdatatypes.Canonical{
							Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export",
						},
					},
					{
						Name: &fhirdatatypes.String{Value: "group-export"},
						Definition: &fhirdatatypes.Canonical{
							Value: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export",
						},
					},
				},
			},
		},
	}
	return statement
}
func WriteCapabilityStatement(statement *fhirmodelCS.CapabilityStatement, w http.ResponseWriter) {
	resource := &fhirmodelCR.ContainedResource{
		OneofResource: &fhirmodelCR.ContainedResource_CapabilityStatement{CapabilityStatement: statement},
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

func WriteBundleResponse(bundle *fhirmodelCR.Bundle, w http.ResponseWriter) {
	resource := &fhirmodelCR.ContainedResource{
		OneofResource: &fhirmodelCR.ContainedResource_Bundle{Bundle: bundle},
	}
	bundleJSON, err := marshaller.Marshal(resource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bundleJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
