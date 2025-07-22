package responseutils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

type ResponseWriter struct {
	marshaller *jsonformat.Marshaller
}

func NewResponseWriter() ResponseWriter {
	// Ensure that we write the serialized FHIR resources as a single line.
	// Needed to comply with the NDJSON format that we are using.
	marshaller, err := jsonformat.NewMarshaller(false, "", "", fhirversion.STU3)
	if err != nil {
		log.API.Fatalf("Failed to create marshaller %s", err)
	}
	return ResponseWriter{marshaller: marshaller}
}

func (r ResponseWriter) Exception(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r ResponseWriter) NotFound(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_NOT_FOUND, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r ResponseWriter) JobsBundle(ctx context.Context, w http.ResponseWriter, jobs []*models.Job, host string) {
	jb := r.CreateJobsBundle(jobs, host)
	r.WriteBundleResponse(jb, w)
}

func (r ResponseWriter) CreateJobsBundle(jobs []*models.Job, host string) *fhirmodels.Bundle {
	var entries []*fhirmodels.Bundle_Entry

	// generate bundle task entries
	for _, job := range jobs {
		entry := r.CreateJobsBundleEntry(job, host)
		entries = append(entries, entry)
	}

	jobLength, err := safecast.ToUint32(len(jobs))

	if err != nil {
		log.API.Errorln(err)
	}

	return &fhirmodels.Bundle{
		Type:  &fhircodes.BundleTypeCode{Value: fhircodes.BundleTypeCode_SEARCHSET},
		Total: &fhirdatatypes.UnsignedInt{Value: jobLength},
		Entry: entries,
	}
}

func (r ResponseWriter) CreateJobsBundleEntry(job *models.Job, host string) *fhirmodels.Bundle_Entry {
	fhirStatusCode := r.GetFhirStatusCode(job.Status)

	return &fhirmodels.Bundle_Entry{
		Resource: &fhirmodels.ContainedResource{
			OneofResource: &fhirmodels.ContainedResource_Task{
				Task: &fhirmodels.Task{
					Identifier: []*fhirdatatypes.Identifier{
						{
							Use:    &fhirdatatypes.IdentifierUseCode{Value: fhirdatatypes.IdentifierUseCode_OFFICIAL},
							System: &fhirdatatypes.Uri{Value: host + "/api/v1/jobs"},
							Value:  &fhirdatatypes.String{Value: fmt.Sprint(job.ID)},
						},
					},
					Status: &fhircodes.TaskStatusCode{Value: fhirStatusCode},
					Intent: &fhircodes.RequestIntentCode{Value: fhircodes.RequestIntentCode_ORDER},
					Input: []*fhirmodels.Task_Parameter{
						{
							Type:  &fhirdatatypes.CodeableConcept{Text: &fhirdatatypes.String{Value: "BULK FHIR Export"}},
							Value: &fhirmodels.Task_Parameter_Value{Value: &fhirmodels.Task_Parameter_Value_StringValue{StringValue: &fhirdatatypes.String{Value: "GET " + job.RequestURL}}},
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

func (r ResponseWriter) GetFhirStatusCode(status models.JobStatus) fhircodes.TaskStatusCode_Value {
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

func (r ResponseWriter) CreateOpOutcome(severity fhircodes.IssueSeverityCode_Value, code fhircodes.IssueTypeCode_Value,
	errType, diagnostics string) *fhirmodels.OperationOutcome {

	return &fhirmodels.OperationOutcome{
		Issue: []*fhirmodels.OperationOutcome_Issue{
			{
				Severity:    &fhircodes.IssueSeverityCode{Value: severity},
				Code:        &fhircodes.IssueTypeCode{Value: code},
				Diagnostics: &fhirdatatypes.String{Value: diagnostics},
			},
		},
	}
}

func (r ResponseWriter) WriteError(ctx context.Context, outcome *fhirmodels.OperationOutcome, w http.ResponseWriter, code int) {
	logger := log.GetCtxLogger(ctx)
	w.Header().Set(constants.ContentType, constants.FHIRJsonContentType)
	if code == http.StatusServiceUnavailable {
		includeRetryAfterHeader(w)
	}
	w.WriteHeader(code)
	_, err := r.WriteOperationOutcome(w, outcome)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func includeRetryAfterHeader(w http.ResponseWriter) {
	//default retrySeconds: 1 second (may convert to environmental variable later)
	retrySeconds := strconv.FormatInt(int64(1), 10)
	w.Header().Set("Retry-After", retrySeconds)
}

func (r ResponseWriter) WriteOperationOutcome(w io.Writer, outcome *fhirmodels.OperationOutcome) (int, error) {
	resource := &fhirmodels.ContainedResource{
		OneofResource: &fhirmodels.ContainedResource_OperationOutcome{OperationOutcome: outcome},
	}
	outcomeJSON, err := r.marshaller.Marshal(resource)
	if err != nil {
		return -1, err
	}

	return w.Write(outcomeJSON)
}

func (r ResponseWriter) CreateCapabilityStatement(reldate time.Time, relversion, baseurl string) *fhirmodels.CapabilityStatement {
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
			{Value: constants.JsonContentType},
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
func (r ResponseWriter) WriteCapabilityStatement(ctx context.Context, statement *fhirmodels.CapabilityStatement, w http.ResponseWriter) {
	resource := &fhirmodels.ContainedResource{
		OneofResource: &fhirmodels.ContainedResource_CapabilityStatement{CapabilityStatement: statement},
	}
	statementJSON, err := r.marshaller.Marshal(resource)
	if err != nil {
		log.API.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(statementJSON)
	if err != nil {
		log.API.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (r ResponseWriter) WriteBundleResponse(bundle *fhirmodels.Bundle, w http.ResponseWriter) {
	resource := &fhirmodels.ContainedResource{
		OneofResource: &fhirmodels.ContainedResource_Bundle{Bundle: bundle},
	}
	resourceJSON, err := r.marshaller.Marshal(resource)
	if err != nil {
		log.API.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(resourceJSON)
	if err != nil {
		log.API.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
