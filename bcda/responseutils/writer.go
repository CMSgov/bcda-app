package responseutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/stu3"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
)

type FhirResponseWriter struct{}

func NewFhirResponseWriter() FhirResponseWriter {
	return FhirResponseWriter{}
}

func (r FhirResponseWriter) Exception(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(stu3.IssueSeverityError, stu3.IssueTypeCodeException, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) NotFound(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(stu3.IssueSeverityError, stu3.IssueTypeCodeNotFound, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) OpOutcome(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	respStatusToFHIRStatusMap := map[int]stu3.IssueTypeCode{
		400: stu3.IssueTypeCodeStructure,
		401: stu3.IssueTypeCodeForbidden,
		403: stu3.IssueTypeCodeForbidden,
		410: stu3.IssueTypeCodeNotFound,
	}
	oo := r.CreateOpOutcome(stu3.IssueSeverityError, respStatusToFHIRStatusMap[statusCode], errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) JobsBundle(ctx context.Context, w http.ResponseWriter, jobs []*models.Job, host string) {
	jb := r.CreateJobsBundle(jobs, host)
	r.WriteBundleResponse(jb, w)
}

func (r FhirResponseWriter) CreateJobsBundle(jobs []*models.Job, host string) *stu3.Bundle {
	var entries []stu3.BundleEntry

	// generate bundle task entries
	for _, job := range jobs {
		entry := r.CreateJobsBundleEntry(job, host)
		entries = append(entries, *entry)
	}

	jobLength, err := safecast.ToUint32(len(jobs))
	if err != nil {
		log.API.Errorln(err)
	}

	return &stu3.Bundle{
		ResourceType: "Bundle",
		Type:         "searchset",
		Total:        jobLength,
		Entry:        entries,
	}
}

func (r FhirResponseWriter) CreateJobsBundleEntry(job *models.Job, host string) *stu3.BundleEntry {
	fhirStatusCode := r.GetFhirStatusCode(job.Status)

	return &stu3.BundleEntry{
		Resource: &stu3.Task{
			ResourceType: "Task",
			Identifier: []stu3.Identifier{
				{
					Use:    "official",
					System: host + "/api/v1/jobs",
					Value:  fmt.Sprint(job.ID),
				},
			},
			Status: stu3.TaskStatus(fhirStatusCode),
			Intent: stu3.TaskIntentOrder,
			Input: []stu3.Parameter{
				{
					Type: stu3.CodeableConcept{
						Text: "BULK FHIR Export",
					},
					ValueString: "GET " + job.RequestURL,
				},
			},
			ExecutionPeriod: stu3.Period{
				Start: job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				End:   job.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			},
		},
	}
}

func (r FhirResponseWriter) GetFhirStatusCode(status models.JobStatus) stu3.TaskStatus {
	switch status {
	case models.JobStatusFailed, models.JobStatusFailedExpired:
		return stu3.TaskStatusFailed
	case models.JobStatusPending:
		return stu3.TaskStatusAccepted
	case models.JobStatusInProgress:
		return stu3.TaskStatusInProgress
	case models.JobStatusCompleted:
		return stu3.TaskStatusCompleted
	case models.JobStatusArchived, models.JobStatusExpired:
		return stu3.TaskStatusCompleted // fhir task status does not have an equivalent to `expired` or `archived`
	case models.JobStatusCancelled, models.JobStatusCancelledExpired:
		return stu3.TaskStatusCancelled
	}
	return ""
}

func (r FhirResponseWriter) CreateOpOutcome(severity stu3.IssueSeverityCode, code stu3.IssueTypeCode, errType, diagnostics string) *stu3.OperationOutcome {
	return &stu3.OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue: []stu3.Issue{
			{
				Severity:    severity,
				Code:        code,
				Diagnostics: diagnostics,
			},
		},
	}
}

func (r FhirResponseWriter) WriteError(ctx context.Context, outcome *stu3.OperationOutcome, w http.ResponseWriter, code int) {
	logger := log.GetCtxLogger(ctx)
	w.Header().Set(constants.ContentType, constants.FHIRJsonContentType)
	if code == http.StatusServiceUnavailable {
		includeRetryAfterHeader(w)
	}
	w.WriteHeader(code)
	_, err := r.WriteOperationOutcome(w, outcome)
	if err != nil {
		logger.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func includeRetryAfterHeader(w http.ResponseWriter) {
	//default retrySeconds: 1 second (may convert to environmental variable later)
	retrySeconds := strconv.FormatInt(int64(1), 10)
	w.Header().Set("Retry-After", retrySeconds)
}

func (r FhirResponseWriter) WriteOperationOutcome(w io.Writer, outcome *stu3.OperationOutcome) (int, error) {
	outcomeJSON, err := json.Marshal(outcome)
	if err != nil {
		return -1, err
	}
	return w.Write(outcomeJSON)
}

func (r FhirResponseWriter) CreateCapabilityStatement(reldate time.Time, relversion, baseurl string) *stu3.CapabilityStatement {
	bbServer := conf.GetEnv("BB_SERVER_LOCATION")
	statement := &stu3.CapabilityStatement{
		ResourceType: "CapabilityStatement",
		Status:       stu3.PublicationStatusActive,
		Date:         reldate.UTC().Format("2006-01-02T15:04:05Z"),
		Publisher:    constants.PublisherName,
		Kind:         stu3.CapabilityStatementKindInstance,
		Instantiates: []string{
			bbServer + "/baseDstu3/metadata/",
			"http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data",
		},
		Software: stu3.Software{
			Name:        constants.SoftwareName,
			Version:     relversion,
			ReleaseDate: reldate.UTC().Format("2006-01-02T15:04:05Z"),
		},
		Implementation: stu3.Implementation{
			Description: constants.SoftwareDescription,
			Url:         baseurl,
		},
		FhirVersion:   "3.0.1",
		AcceptUnknown: stu3.UnknownContentCodeExtensions,
		Format: []string{
			constants.JsonContentType,
			constants.FHIRJsonContentType,
		},
		Rest: []stu3.CapabilityStatementRest{
			{
				Mode: stu3.RestfulCapabilityModeServer,
				Security: &stu3.Security{
					Cors: true,
					Service: []stu3.CodeableConcept{
						{
							Coding: []stu3.Coding{
								{
									Display: "OAuth",
									Code:    "OAuth",
									System:  constants.RestfulSecurityServiceSystem,
								},
							},
							Text: "OAuth",
						},
					},
					Extension: []stu3.Extension{
						{
							Url: constants.SmartOAuthURIsExtensionURL,
							Extension: []stu3.Extension{
								{
									Url:      "token",
									ValueUri: baseurl + "/auth/token",
								},
							},
						},
					},
				},
				Interaction: []stu3.Interaction{
					{
						Code: stu3.SystemRestfulInteractionBatch,
					},
					{
						Code: stu3.SystemRestfulInteractionSearchSystem,
					},
				},
				Operation: []stu3.RestOperation{
					{
						Name: "patient-export",
						Definition: &stu3.Reference{
							Reference: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export",
						},
					},
					{
						Name: "group-export",
						Definition: &stu3.Reference{
							Reference: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export",
						},
					},
				},
			},
		},
	}
	return statement
}

func (r FhirResponseWriter) WriteCapabilityStatement(ctx context.Context, statement *stu3.CapabilityStatement, w http.ResponseWriter) {
	statementJSON, err := json.Marshal(statement)
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

func (r FhirResponseWriter) WriteBundleResponse(bundle *stu3.Bundle, w http.ResponseWriter) {
	resourceJSON, err := json.Marshal(bundle)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(resourceJSON)
	if err != nil {
		log.API.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
