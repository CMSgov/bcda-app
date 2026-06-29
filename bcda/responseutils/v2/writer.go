package responseutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/r4"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
)

type FhirResponseWriter struct{}

func NewFhirResponseWriter() FhirResponseWriter {
	return FhirResponseWriter{}
}

func (r FhirResponseWriter) Exception(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(r4.IssueSeverityError, r4.IssueTypeCodeException, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) NotFound(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	oo := r.CreateOpOutcome(r4.IssueSeverityError, r4.IssueTypeCodeNotFound, errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) OpOutcome(ctx context.Context, w http.ResponseWriter, statusCode int, errType, errMsg string) {
	respStatusToFHIRStatusMap := map[int]r4.IssueTypeCode{
		400: r4.IssueTypeCodeStructure,
		401: r4.IssueTypeCodeForbidden,
		403: r4.IssueTypeCodeForbidden,
		410: r4.IssueTypeCodeNotFound,
	}
	oo := r.CreateOpOutcome(r4.IssueSeverityError, respStatusToFHIRStatusMap[statusCode], errType, errMsg)
	r.WriteError(ctx, oo, w, statusCode)
}

func (r FhirResponseWriter) JobsBundle(ctx context.Context, w http.ResponseWriter, jobs []*models.Job, host string) {
	jb := r.CreateJobsBundle(jobs, host)
	r.WriteBundleResponse(jb, w)
}

func (r FhirResponseWriter) CreateJobsBundle(jobs []*models.Job, host string) *r4.Bundle {
	var entries []r4.BundleEntry

	// generate bundle task entries
	for _, job := range jobs {
		entry := r.CreateJobsBundleEntry(job, host)
		entries = append(entries, *entry)
	}

	jobLength, err := safecast.ToUint32(len(jobs))
	if err != nil {
		log.API.Errorln(err)
	}

	return &r4.Bundle{
		ResourceType: "Bundle",
		Type:         "searchset",
		Total:        jobLength,
		Entry:        entries,
	}
}

func (r FhirResponseWriter) CreateJobsBundleEntry(job *models.Job, host string) *r4.BundleEntry {
	fhirStatusCode := r.GetFhirStatusCode(job.Status)

	return &r4.BundleEntry{
		Resource: &r4.Task{
			ResourceType: "Task",
			Identifier: []r4.Identifier{
				{
					Use:    "official",
					System: host + "/api/v2/jobs",
					Value:  fmt.Sprint(job.ID),
				},
			},
			Status: r4.TaskStatus(fhirStatusCode),
			Intent: r4.TaskIntentOrder,
			Input: []r4.Parameter{
				{
					Type: r4.CodeableConcept{
						Text: "BULK FHIR Export",
					},
					Value: "GET " + job.RequestURL,
				},
			},
			ExecutionPeriod: r4.Period{
				Start: job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				End:   job.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			},
		},
	}
}

func (r FhirResponseWriter) GetFhirStatusCode(status models.JobStatus) r4.TaskStatus {
	switch status {
	case models.JobStatusFailed, models.JobStatusFailedExpired:
		return r4.TaskStatusFailed
	case models.JobStatusPending:
		return r4.TaskStatusAccepted
	case models.JobStatusInProgress:
		return r4.TaskStatusInProgress
	case models.JobStatusCompleted:
		return r4.TaskStatusCompleted
	case models.JobStatusArchived, models.JobStatusExpired:
		return r4.TaskStatusCompleted // fhir task status does not have an equivalent to `expired` or `archived`
	case models.JobStatusCancelled, models.JobStatusCancelledExpired:
		return r4.TaskStatusCancelled
	}
	return ""
}

func (r FhirResponseWriter) CreateOpOutcome(severity r4.IssueSeverityCode, code r4.IssueTypeCode, errType, diagnostics string) *r4.OperationOutcome {
	return &r4.OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue: []r4.Issue{
			{
				Severity:    severity,
				Code:        code,
				Diagnostics: diagnostics,
				Details: &r4.CodeableConcept{
					Coding: []r4.Coding{
						{
							System:  "http://hl7.org/fhir/ValueSet/operation-outcome",
							Code:    responseutils.RequestErr,
							Display: diagnostics,
						},
					},
					Text: diagnostics,
				},
			},
		},
	}
}

func (r FhirResponseWriter) WriteError(ctx context.Context, outcome *r4.OperationOutcome, w http.ResponseWriter, code int) {
	logger := log.GetCtxLogger(ctx)
	w.Header().Set(constants.ContentType, constants.FHIRJsonContentType)
	w.WriteHeader(code)
	_, err := r.WriteOperationOutcome(w, outcome)
	if err != nil {
		logger.WithField("resp_status", http.StatusInternalServerError).Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (r FhirResponseWriter) WriteOperationOutcome(w io.Writer, outcome *r4.OperationOutcome) (int, error) {
	outcomeJSON, err := json.Marshal(outcome)
	if err != nil {
		return -1, err
	}
	return w.Write(outcomeJSON)
}


func (r FhirResponseWriter) WriteBundleResponse(bundle *r4.Bundle, w http.ResponseWriter) {
	bundleJSON, err := json.Marshal(bundle)
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
