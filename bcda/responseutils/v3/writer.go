package responseutils

import (
	"context"
	"fmt"
	"io"

	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	fhirmodelCR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhirmodelCS "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
	fhirmodelOO "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/operation_outcome_go_proto"
	fhirmodelT "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/task_go_proto"
	fhirvaluesets "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"
)

type FhirResponseWriter struct {
	marshaller *jsonformat.Marshaller
}

func NewFhirResponseWriter() FhirResponseWriter {
	// Ensure that we write the serialized FHIR resources as a single line.
	// Needed to comply with the NDJSON format that we are using.
	m, err := jsonformat.NewMarshaller(false, "", "", fhirversion.R4)
	if err != nil {
		log.API.Fatalf("Failed to create marshaller %s", err)
	}
	return FhirResponseWriter{marshaller: m}
}

func (r FhirResponseWriter) Exception(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) NotFound(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_NOT_FOUND, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) OpOutcome(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	respStatusToFHIRStatusMap := map[int]fhircodes.IssueTypeCode_Value{
		400: fhircodes.IssueTypeCode_STRUCTURE,
		401: fhircodes.IssueTypeCode_FORBIDDEN,
		403: fhircodes.IssueTypeCode_FORBIDDEN,
		410: fhircodes.IssueTypeCode_NOT_FOUND,
	}
	oo := r.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, respStatusToFHIRStatusMap[statusCode], errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) JobsBundle(ctx context.Context, w http.ResponseWriter, jobs []*models.Job, host string) {
	jb := r.CreateJobsBundle(jobs, host)
	r.WriteBundleResponse(jb, w)
}

func (r FhirResponseWriter) CreateJobsBundle(jobs []*models.Job, host string) *fhirmodelCR.Bundle {
	var entries []*fhirmodelCR.Bundle_Entry

	// generate bundle task entries
	for _, job := range jobs {
		entry := r.CreateJobsBundleEntry(job, host)
		entries = append(entries, entry)
	}

	jobLength, err := safecast.ToUint32(len(jobs))

	if err != nil {
		log.API.Errorln(err)
	}

	return &fhirmodelCR.Bundle{
		Type:  &fhirmodelCR.Bundle_TypeCode{Value: fhircodes.BundleTypeCode_SEARCHSET},
		Total: &fhirdatatypes.UnsignedInt{Value: jobLength},
		Entry: entries,
	}
}

func (r FhirResponseWriter) CreateJobsBundleEntry(job *models.Job, host string) *fhirmodelCR.Bundle_Entry {
	fhirStatusCode := r.GetFhirStatusCode(job.Status)

	return &fhirmodelCR.Bundle_Entry{
		Resource: &fhirmodelCR.ContainedResource{
			OneofResource: &fhirmodelCR.ContainedResource_Task{
				Task: &fhirmodelT.Task{
					Identifier: []*fhirdatatypes.Identifier{
						{
							Use:    &fhirdatatypes.Identifier_UseCode{Value: fhircodes.IdentifierUseCode_OFFICIAL},
							System: &fhirdatatypes.Uri{Value: host + "/api/v3/jobs"},
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

func (r FhirResponseWriter) GetFhirStatusCode(status models.JobStatus) fhircodes.TaskStatusCode_Value {
	var fhirStatus fhircodes.TaskStatusCode_Value

	switch status {

	case models.JobStatusFailed, models.JobStatusFailedExpired:
		fhirStatus = fhircodes.TaskStatusCode_FAILED
	case models.JobStatusPending:
		fhirStatus = fhircodes.TaskStatusCode_ACCEPTED
	case models.JobStatusInProgress:
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

func (r FhirResponseWriter) CreateOpOutcome(severity fhircodes.IssueSeverityCode_Value, code fhircodes.IssueTypeCode_Value,
	errType, diagnostics string) *fhirmodelOO.OperationOutcome {

	return &fhirmodelOO.OperationOutcome{
		Issue: []*fhirmodelOO.OperationOutcome_Issue{
			{
				Severity:    &fhirmodelOO.OperationOutcome_Issue_SeverityCode{Value: severity},
				Code:        &fhirmodelOO.OperationOutcome_Issue_CodeType{Value: code},
				Diagnostics: &fhirdatatypes.String{Value: diagnostics},
				Details: &fhirdatatypes.CodeableConcept{
					// Coding is intentionally omitted for v3 to conform with FHIR base profile
					Text: &fhirdatatypes.String{Value: diagnostics},
				},
			},
		},
	}
}

func (r FhirResponseWriter) WriteError(ctx context.Context, outcome *fhirmodelOO.OperationOutcome, w http.ResponseWriter, code int) {
	//Write application/fhir+json header on OperationOutcome responses
	//https://build.fhir.org/ig/HL7/bulk-data/export.html#response---error-status-1
	logger := log.GetCtxLogger(ctx)
	w.Header().Set(constants.ContentType, constants.FHIRJsonContentType)
	w.WriteHeader(code)
	_, err := r.WriteOperationOutcome(w, outcome)
	if err != nil {
		logger.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (r FhirResponseWriter) WriteOperationOutcome(w io.Writer, outcome *fhirmodelOO.OperationOutcome) (int, error) {
	resource := &fhirmodelCR.ContainedResource{
		OneofResource: &fhirmodelCR.ContainedResource_OperationOutcome{OperationOutcome: outcome},
	}
	outcomeJSON, err := r.marshaller.Marshal(resource)
	if err != nil {
		return -1, err
	}

	return w.Write(outcomeJSON)
}

func (r FhirResponseWriter) CreateCapabilityStatement(reldate time.Time, relversion, baseurl string) *fhirmodelCS.CapabilityStatement {
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
			{Value: constants.JsonContentType},
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
func (r FhirResponseWriter) WriteCapabilityStatement(ctx context.Context, statement *fhirmodelCS.CapabilityStatement, w http.ResponseWriter) {
	resource := &fhirmodelCR.ContainedResource{
		OneofResource: &fhirmodelCR.ContainedResource_CapabilityStatement{CapabilityStatement: statement},
	}
	statementJSON, err := r.marshaller.Marshal(resource)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(statementJSON)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (r FhirResponseWriter) WriteBundleResponse(bundle *fhirmodelCR.Bundle, w http.ResponseWriter) {
	resource := &fhirmodelCR.ContainedResource{
		OneofResource: &fhirmodelCR.ContainedResource_Bundle{Bundle: bundle},
	}
	bundleJSON, err := r.marshaller.Marshal(resource)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bundleJSON)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
