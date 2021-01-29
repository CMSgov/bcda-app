package v1

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/conf"
)

var h *api.Handler

func init() {
	h = api.NewHandler([]string{"Patient", "Coverage", "ExplanationOfBenefit"}, "/v1/fhir")
}

/*
	swagger:route GET /api/v1/Patient/$export bulkData bulkPatientRequest

	Start FHIR STU3 data export for all supported resource types

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
	swagger:route GET /api/v1/Group/{groupId}/$export bulkData bulkGroupRequest

    Start FHIR STU3 data export (for the specified group identifier) for all supported resource types

	Initiates a job to collect data from the Blue Button API for your ACO. The supported Group identifiers are `all` and `runout`.

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
	swagger:route GET /api/v1/jobs/{jobId} job jobStatus

	Get job status

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
func JobStatus(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, err.Error())
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	job, jobKeys, err := h.Svc.GetJobAndKeys(context.Background(), uint(jobID))
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.DbErr, "")
		// NOTE: This is a catch all and may not necessarily mean that the job was not found.
		// So returning a StatusNotFound may be a misnomer
		responseutils.WriteError(oo, w, http.StatusNotFound)
		return
	}

	switch job.Status {

	case models.JobStatusFailed, models.JobStatusFailedExpired:
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.InternalErr, "Service encountered numerous errors.  Unable to complete the request.")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
	case models.JobStatusPending, models.JobStatusInProgress:
		w.Header().Set("X-Progress", job.StatusMessage())
		w.WriteHeader(http.StatusAccepted)
		return
	case models.JobStatusCompleted:
		// If the job should be expired, but the cleanup job hasn't run for some reason, still respond with 410
		if job.UpdatedAt.Add(api.GetJobTimeout()).Before(time.Now()) {
			w.Header().Set("Expires", job.UpdatedAt.Add(api.GetJobTimeout()).String())
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, "")
			responseutils.WriteError(oo, w, http.StatusGone)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Expires", job.UpdatedAt.Add(api.GetJobTimeout()).String())
		scheme := "http"
		if servicemux.IsHTTPS(r) {
			scheme = "https"
		}

		rb := api.BulkResponseBody{
			TransactionTime:     job.TransactionTime,
			RequestURL:          job.RequestURL,
			RequiresAccessToken: true,
			Files:               []api.FileItem{},
			Errors:              []api.FileItem{},
			JobID:               job.ID,
		}

		for _, jobKey := range jobKeys {
			// data files
			fi := api.FileItem{
				Type: jobKey.ResourceType,
				URL:  fmt.Sprintf("%s://%s/data/%d/%s", scheme, r.Host, jobID, strings.TrimSpace(jobKey.FileName)),
			}
			rb.Files = append(rb.Files, fi)

			// error files
			errFileName := strings.Split(jobKey.FileName, ".")[0]
			errFilePath := fmt.Sprintf("%s/%d/%s-error.ndjson", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID, errFileName)
			if _, err := os.Stat(errFilePath); !os.IsNotExist(err) {
				errFI := api.FileItem{
					Type: "OperationOutcome",
					URL:  fmt.Sprintf("%s://%s/data/%d/%s-error.ndjson", scheme, r.Host, jobID, errFileName),
				}
				rb.Errors = append(rb.Errors, errFI)
			}
		}

		jsonData, err := json.Marshal(rb)
		if err != nil {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	case models.JobStatusArchived, models.JobStatusExpired:
		w.Header().Set("Expires", job.UpdatedAt.Add(api.GetJobTimeout()).String())
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
			responseutils.NotFoundErr, "")
		responseutils.WriteError(oo, w, http.StatusGone)
	case models.JobStatusCancelled:
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_NOT_FOUND,
			responseutils.NotFoundErr, "Job has been cancelled.")
		responseutils.WriteError(oo, w, http.StatusNotFound)
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

/*
	swagger:route DELETE /api/v1/jobs/{jobId} job deleteJob

	Cancel a job

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
func DeleteJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, err.Error())
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	updatedJobID, err := h.Svc.CancelJob(context.Background(), uint(jobID))
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	if updatedJobID == uint(jobID) {
		w.WriteHeader(http.StatusAccepted)
	} else {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Deleted, "Unable to cancel jobs not currently in progress or pending.")
		responseutils.WriteError(oo, w, http.StatusGone)
	}
}

/*
	swagger:route GET /data/{jobId}/{filename} job serveData

	Get data file

	Returns the NDJSON file of data generated by an export job.  Will be in the format <UUID>.ndjson.  Get the full value from the job status response

	Produces:
	- application/fhir+json

	Schemes: http, https

	Security:
		bearer_token:

	Responses:
		200: FileNDJSON
		400: badRequestResponse
		401: invalidCredentials
		404: notFoundResponse
		500: errorResponse
*/
func ServeData(w http.ResponseWriter, r *http.Request) {
	dataDir := conf.GetEnv("FHIR_PAYLOAD_DIR")
	fileName := chi.URLParam(r, "fileName")
	jobID := chi.URLParam(r, "jobID")
	w.Header().Set("Content-Type", "application/fhir+ndjson")

	var useGZIP bool
	for _, header := range r.Header.Values("Accept-Encoding") {
		if header == "gzip" {
			useGZIP = true
			break
		}
	}

	if useGZIP {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		http.ServeFile(gzw, r, fmt.Sprintf("%s/%s/%s", dataDir, jobID, fileName))
	} else {
		http.ServeFile(w, r, fmt.Sprintf("%s/%s/%s", dataDir, jobID, fileName))
	}
}

/*
	swagger:route GET /api/v1/metadata metadata metadata

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

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	host := fmt.Sprintf("%s://%s", scheme, r.Host)
	statement := responseutils.CreateCapabilityStatement(dt, constants.Version, host)
	responseutils.WriteCapabilityStatement(statement, w)
}

/*
	swagger:route GET /_version metadata getVersion

	Get API version

	Returns the version of the API that is currently running. Note that this endpoint is **not** prefixed with the base path (e.g. /api/v1).

	Produces:
	- application/json

	Schemes: http, https

	Responses:
		200: VersionResponse
*/
func GetVersion(w http.ResponseWriter, r *http.Request) {
	respMap := make(map[string]string)
	respMap["version"] = constants.Version
	respBytes, err := json.Marshal(respMap)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.InternalErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respBytes)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.InternalErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]string)

	if health.IsDatabaseOK() {
		m["database"] = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		m["database"] = "error"
		w.WriteHeader(http.StatusBadGateway)
	}

	respJSON, err := json.Marshal(m)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respJSON)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

/*
   swagger:route GET /_auth metadata getAuthInfo

   Get details about auth

   Returns the auth provider that is currently being used. Note that this endpoint is **not** prefixed with the base path (e.g. /api/v1).

   Produces:
   - application/json

   Schemes: http, https

   Responses:
           200: AuthResponse
*/
func GetAuthInfo(w http.ResponseWriter, r *http.Request) {
	respMap := make(map[string]string)
	respMap["auth_provider"] = auth.GetProviderName()
	version, err := auth.GetProvider().GetVersion()
	if err == nil {
		respMap["version"] = version
	} else {
		respMap["error message"] = err.Error()
	}
	respBytes, err := json.Marshal(respMap)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respBytes)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
