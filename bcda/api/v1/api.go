package v1

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/CMSgov/bcda-app/bcda/api"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

var h *api.Handler

func init() {
	resources, ok := service.GetDataTypes([]string{
		"Patient",
		"Coverage",
		"ExplanationOfBenefit",
		"Observation",
	}...)

	if ok {
		h = api.NewHandler(resources, "/v1/fhir", "v1")
	} else {
		panic("Failed to configure resource DataTypes")
	}
}

/*
swagger:route GET /api/v1/alr/$export alrData alrRequest

# Start FHIR STU3 data export for all supported resource types

Initiates a job to collect Assignment List Report data for your ACO. Supported resource types are Patient, Coverage, Group, Risk Assessment, Observation, and Covid Episode.

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
func ALRRequest(w http.ResponseWriter, r *http.Request) {
	h.ALRRequest(w, r)
}

/*
swagger:route GET /api/v1/Patient/$export bulkData bulkPatientRequest

# Start FHIR STU3 data export for all supported resource types

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
func JobStatus(w http.ResponseWriter, r *http.Request) {
	h.JobStatus(w, r)
}

/*
swagger:route GET /api/v1/jobs job jobsStatus

# Get jobs statuses

Returns the current statuses of export jobs. Supported status types are Completed, Archived, Expired, Failed, FailedExpired,
Pending, In Progress, Cancelled, and CancelledExpired. If no status(s) is provided, all jobs will be returned.

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
func JobsStatus(w http.ResponseWriter, r *http.Request) {
	h.JobsStatus(w, r)
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	int, err := w.Writer.Write(b)
	if err != nil {
		log.API.Errorf("Error encountered in writing bytes with gzip writer: %s", err.Error())
	}
	return int, err
}

/*
swagger:route DELETE /api/v1/jobs/{jobId} job deleteJob

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
func DeleteJob(w http.ResponseWriter, r *http.Request) {
	h.DeleteJob(w, r)
}

/*
swagger:route GET /api/v1/attribution_status attributionStatus attributionStatus

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
func AttributionStatus(w http.ResponseWriter, r *http.Request) {
	h.AttributionStatus(w, r)
}

/*
swagger:route GET /data/{jobId}/{filename} job serveData

# Get data file

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
	filePath := fmt.Sprintf("%s/%s/%s", dataDir, jobID, fileName)

	encoded, err := isGzipEncoded(filePath)
	if err != nil {
		writeServeDataFailure(err, w)
	}

	var useGZIP bool
	for _, header := range r.Header.Values("Accept-Encoding") {
		if strings.Contains(header, "gzip") {
			useGZIP = true
			break
		}
	}
	w.Header().Set(constants.ContentType, "application/fhir+ndjson")
	if useGZIP {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		if encoded {
			http.ServeFile(w, r, filePath)
		} else {
			http.ServeFile(gzw, r, filePath)
		}

	} else {
		log.API.Warnf("API request to serve data is being made without gzip for file %s for jobId %s", fileName, jobID)
		if encoded {
			//We'll do the following: 1. Open file, 2. De-compress it, 3. Serve it up.
			file, err := os.Open(filePath) // #nosec G304
			if err != nil {
				writeServeDataFailure(err, w)
				return
			}
			defer file.Close() //#nosec G307
			gzipReader, err := gzip.NewReader(file)
			if err != nil {
				writeServeDataFailure(err, w)
				return
			}
			defer gzipReader.Close()
			_, err = io.Copy(w, gzipReader) // #nosec G110
			if err != nil {
				writeServeDataFailure(err, w)
				return
			}
		} else {
			http.ServeFile(w, r, filePath)
		}
	}
}

// This function is not necessary, but helps meet the sonarQube quality gates
func writeServeDataFailure(err error, w http.ResponseWriter) {
	log.API.Error(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// This function reads a file's magic number, to determine if it is gzip-encoded or not.
func isGzipEncoded(filePath string) (encoded bool, err error) {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return false, err
	}
	defer file.Close() //#nosec G307

	byteSlice := make([]byte, 2)
	bytesRead, err := file.Read(byteSlice)
	if err != nil {
		return false, err
	}

	//We can't compare to a magic number if there's less than 2 bytes returned. Also can't be ndjson
	if bytesRead < 2 {
		return false, errors.New("Invalid file with length 1 byte.")
	}

	comparison := []byte{0x1f, 0x8b}
	return bytes.Equal(comparison, byteSlice), nil
}

/*
swagger:route GET /api/v1/metadata metadata metadata

# Get metadata

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
	responseutils.WriteCapabilityStatement(r.Context(), statement, w)
}

/*
swagger:route GET /_version metadata getVersion

# Get API version

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
		log.API.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	_, err = w.Write(respBytes)
	if err != nil {
		log.API.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]string)

	dbStatus, dbOK := health.IsDatabaseOK()
	ssasStatus, ssasOK := health.IsSsasOK()

	m["database"] = dbStatus
	m["ssas"] = ssasStatus

	if !dbOK || !ssasOK {
		w.WriteHeader(http.StatusBadGateway)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	respJSON, err := json.Marshal(m)
	if err != nil {
		log.API.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	_, err = w.Write(respJSON)
	if err != nil {
		log.API.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

/*
swagger:route GET /_auth metadata getAuthInfo

# Get details about auth

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
		logger := log.GetCtxLogger(r.Context())
		logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set(constants.ContentType, constants.JsonContentType)
	_, err = w.Write(respBytes)
	if err != nil {
		logger := log.GetCtxLogger(r.Context())
		logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
