package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/models/postgres"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bgentry/que-go"
	fhirmodels "github.com/eug48/fhir/models"

	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

var (
	qc *que.Client
)

const (
	groupAll = "all"
)

func init() {
	// Ensure that models.go is properly initialized with the service reference.
	// As we refactor more of the code, we should be able to remove the initialization
	// from models.go
	cutoffDuration := time.Duration(utils.GetEnvInt("CCLF_CUTOFF_DATE_DAYS", 45)*24) * time.Hour
	db := database.GetGORMDbConnection()
	db.DB().SetMaxOpenConns(utils.GetEnvInt("BCDA_DB_MAX_OPEN_CONNS", 25))
	db.DB().SetMaxIdleConns(utils.GetEnvInt("BCDA_DB_MAX_IDLE_CONNS", 25))
	db.DB().SetConnMaxLifetime(time.Duration(utils.GetEnvInt("BCDA_DB_CONN_MAX_LIFETIME_MIN", 5)) * time.Minute)
	repository := postgres.NewRepository(db)
	models.GetService(repository, cutoffDuration, utils.GetEnvInt("BCDA_SUPPRESSION_LOOKBACK_DAYS", 60))
}

/*
	swagger:route GET /api/v1/Patient/$export bulkData bulkPatientRequest

	Start data export for all supported resource types

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
func bulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	resourceTypes, err := validateRequest(r)
	if err != nil {
		responseutils.WriteError(err, w, http.StatusBadRequest)
		return
	}
	retrieveNewBeneHistData := false // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	bulkRequest(resourceTypes, w, r, retrieveNewBeneHistData)
}

/*
	swagger:route GET /api/v1/Group/{groupId}/$export bulkData bulkGroupRequest

    Start data export (for the specified group identifier) for all supported resource types

	Initiates a job to collect data from the Blue Button API for your ACO. The only Group identifier supported by the system is `all`.  The `all` identifier returns data for the group of all patients attributed to the requesting ACO.  If used when specifying `_since`: all claims data which has been updated since the specified date will be returned for beneficiaries which have been attributed to the ACO since before the specified date; and all historical claims data will be returned for beneficiaries which have been newly attributed to the ACO since the specified date.

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
func bulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	retrieveNewBeneHistData := false

	groupID := chi.URLParam(r, "groupId")
	if groupID == groupAll {
		resourceTypes, err := validateRequest(r)
		if err != nil {
			responseutils.WriteError(err, w, http.StatusBadRequest)
			return
		}

		// Set flag to retrieve new beneficiaries' historical data if _since param is provided and feature is turned on
		_, ok := r.URL.Query()["_since"]
		if ok && utils.GetEnvBool("BCDA_ENABLE_NEW_GROUP", false) {
			retrieveNewBeneHistData = true
		}

		bulkRequest(resourceTypes, w, r, retrieveNewBeneHistData)
	} else {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Invalid group ID")
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}
}

func bulkRequest(resourceTypes []string, w http.ResponseWriter, r *http.Request, retrieveNewBeneHistData bool) {
	var (
		ad  auth.AuthData
		err error
	)

	if ad, err = readAuthData(r); err != nil {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.TokenErr, "")
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	acoID := ad.ACOID

	var jobs []models.Job
	// If we really do find this record with the below matching criteria then this particular ACO has already made
	// a bulk data request and it has yet to finish. Users will be presented with a 429 Too-Many-Requests error until either
	// their job finishes or time expires (+24 hours default) for any remaining jobs left in a pending or in-progress state.
	// Overall, this will prevent a queue of concurrent calls from slowing up our system.
	// NOTE: this logic is relevant to PROD only; simultaneous requests in our lower environments is acceptable (i.e., shared opensbx creds)
	if (os.Getenv("DEPLOYMENT_TARGET") == "prod") && (!db.Find(&jobs, "aco_id = ?", acoID).RecordNotFound()) {
		if types, ok := check429(jobs, resourceTypes, w); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(utils.GetEnvInt("CLIENT_RETRY_AFTER_IN_SECONDS", 0)))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		} else {
			resourceTypes = types
		}
	}

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	newJob := models.Job{
		ACOID:      uuid.Parse(acoID),
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     "Pending",
	}
	if result := db.Save(&newJob); result.Error != nil {
		log.Error(result.Error.Error())
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	bb, err := client.NewBlueButtonClient()
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	// request a fake patient in order to acquire the bundle's lastUpdated metadata
	jsonData, err := bb.GetPatient("FAKE_PATIENT", strconv.FormatUint(uint64(newJob.ID), 10), acoID, "", time.Now())
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "Failure to retrieve transactionTime metadata from FHIR Data Server.")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	var patient models.Patient
	err = json.Unmarshal([]byte(jsonData), &patient)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "Failure to parse transactionTime metadata from FHIR Data Server.")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	transactionTime := patient.Meta.LastUpdated
	if db.Model(&newJob).Update("transaction_time", transactionTime).Error != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	if qc == nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// Decode the _since parameter (if it exists) so it can be persisted in job args
	var decodedSince string
	params, ok := r.URL.Query()["_since"]
	if ok {
		decodedSince, _ = url.QueryUnescape(params[0])
	}

	var enqueueJobs []*que.Job
	enqueueJobs, err = newJob.GetEnqueJobs(resourceTypes, decodedSince, retrieveNewBeneHistData)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	if db.Model(&newJob).Update("job_count", len(enqueueJobs)).Error != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	for _, j := range enqueueJobs {
		if err = qc.Enqueue(j); err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/v1/jobs/%d", scheme, r.Host, newJob.ID))
	w.WriteHeader(http.StatusAccepted)
}

func check429(jobs []models.Job, types []string, w http.ResponseWriter) ([]string, bool) {
	var unworkedTypes []string

	for _, t := range types {
		worked := false
		for _, job := range jobs {
			req, err := url.Parse(job.RequestURL)
			if err != nil {
				log.Error(err)
				oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
				responseutils.WriteError(oo, w, http.StatusInternalServerError)
			}

			if requestedTypes, ok := req.Query()["_type"]; ok {
				// if this type is being worked no need to keep looking, break out and go to the next type.
				if strings.Contains(requestedTypes[0], t) && (job.Status == "Pending" || job.Status == "In Progress") && (job.CreatedAt.Add(GetJobTimeout()).After(time.Now())) {
					worked = true
					break
				}
			} else {
				// check to see if the export all is still being worked
				if (job.Status == "Pending" || job.Status == "In Progress") && (job.CreatedAt.Add(GetJobTimeout()).After(time.Now())) {
					return nil, false
				}
			}
		}
		if !worked {
			unworkedTypes = append(unworkedTypes, t)
		}
	}
	if len(unworkedTypes) == 0 {
		return nil, false
	} else {
		return unworkedTypes, true
	}
}

func validateRequest(r *http.Request) ([]string, *fhirmodels.OperationOutcome) {

	// validate optional "_type" parameter
	var resourceTypes []string
	params, ok := r.URL.Query()["_type"]
	if ok {
		resourceMap := make(map[string]bool)
		params = strings.Split(params[0], ",")
		for _, p := range params {
			if p != "ExplanationOfBenefit" && p != "Patient" && p != "Coverage" {
				oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Invalid resource type")
				return nil, oo
			} else {
				if !resourceMap[p] {
					resourceMap[p] = true
					resourceTypes = append(resourceTypes, p)
				} else {
					oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Repeated resource type")
					return nil, oo
				}
			}
		}
	} else {
		// resource types not supplied in request; default to applying all resource types.
		resourceTypes = append(resourceTypes, "Patient", "ExplanationOfBenefit", "Coverage")
	}

	// validate optional "_since" parameter
	params, ok = r.URL.Query()["_since"]
	if ok {
		sinceDate, err := time.Parse(time.RFC3339Nano, params[0])
		if err != nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format.")
			return nil, oo
		} else if sinceDate.After(time.Now()) {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "Invalid date format supplied in _since parameter. Date must be a date that has already passed")
			return nil, oo
		}
	}

	//validate "_outputFormat" parameter
	params, ok = r.URL.Query()["_outputFormat"]
	if ok {
		if params[0] != "ndjson" && params[0] != "application/fhir+ndjson" && params[0] != "application/ndjson" {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "_outputFormat parameter must be application/fhir+ndjson, application/ndjson, or ndjson")
			return nil, oo
		}
	}

	// we do not support "_elements" parameter
	_, ok = r.URL.Query()["_elements"]
	if ok {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Invalid parameter: this server does not support the _elements parameter.")
		return nil, oo
	}

	return resourceTypes, nil
}

/*
	swagger:route GET /api/v1/jobs/{jobId} bulkData jobStatus

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
func jobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var job models.Job
	err := db.Find(&job, "id = ?", jobID).Error
	if err != nil {
		log.Print(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusNotFound)
		return
	}

	switch job.Status {

	case "Failed":
		responseutils.WriteError(&fhirmodels.OperationOutcome{}, w, http.StatusInternalServerError)
	case "Pending":
		fallthrough
	case "In Progress":
		w.Header().Set("X-Progress", job.StatusMessage())
		w.WriteHeader(http.StatusAccepted)
		return
	case "Completed":
		// If the job should be expired, but the cleanup job hasn't run for some reason, still respond with 410
		if job.UpdatedAt.Add(GetJobTimeout()).Before(time.Now()) {
			w.Header().Set("Expires", job.UpdatedAt.Add(GetJobTimeout()).String())
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Deleted, "")
			responseutils.WriteError(oo, w, http.StatusGone)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Expires", job.UpdatedAt.Add(GetJobTimeout()).String())
		scheme := "http"
		if servicemux.IsHTTPS(r) {
			scheme = "https"
		}

		rb := bulkResponseBody{
			TransactionTime:     job.TransactionTime,
			RequestURL:          job.RequestURL,
			RequiresAccessToken: true,
			Files:               []fileItem{},
			Errors:              []fileItem{},
			JobID:               job.ID,
		}

		var jobKeysObj []models.JobKey
		db.Find(&jobKeysObj, "job_id = ?", job.ID)
		for _, jobKey := range jobKeysObj {

			// data files
			fi := fileItem{
				Type: jobKey.ResourceType,
				URL:  fmt.Sprintf("%s://%s/data/%s/%s", scheme, r.Host, jobID, strings.TrimSpace(jobKey.FileName)),
			}
			rb.Files = append(rb.Files, fi)

			// error files
			errFileName := strings.Split(jobKey.FileName, ".")[0]
			errFilePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), jobID, errFileName)
			if _, err := os.Stat(errFilePath); !os.IsNotExist(err) {
				errFI := fileItem{
					Type: "OperationOutcome",
					URL:  fmt.Sprintf("%s://%s/data/%s/%s-error.ndjson", scheme, r.Host, jobID, errFileName),
				}
				rb.Errors = append(rb.Errors, errFI)
			}
		}

		jsonData, err := json.Marshal(rb)
		if err != nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	case "Archived":
		fallthrough
	case "Expired":
		w.Header().Set("Expires", job.UpdatedAt.Add(GetJobTimeout()).String())
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Deleted, "")
		responseutils.WriteError(oo, w, http.StatusGone)
	}
}

/*
	swagger:route GET /data/{jobId}/{filename} bulkData serveData

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
func serveData(w http.ResponseWriter, r *http.Request) {
	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	fileName := chi.URLParam(r, "fileName")
	jobID := chi.URLParam(r, "jobID")
	w.Header().Set("Content-Type", "application/fhir+ndjson")
	http.ServeFile(w, r, fmt.Sprintf("%s/%s/%s", dataDir, jobID, fileName))
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
func metadata(w http.ResponseWriter, r *http.Request) {
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
func getVersion(w http.ResponseWriter, r *http.Request) {
	respMap := make(map[string]string)
	respMap["version"] = constants.Version
	respBytes, err := json.Marshal(respMap)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.InternalErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respBytes)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.InternalErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
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
func getAuthInfo(w http.ResponseWriter, r *http.Request) {
	respMap := make(map[string]string)
	respMap["auth_provider"] = auth.GetProviderName()
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

// swagger:model fileItem
type fileItem struct {
	// FHIR resource type of file contents
	Type string `json:"type"`
	// URL of the file
	URL string `json:"url"`
}

/*
Data export job has completed successfully. The response body will contain a JSON object providing metadata about the transaction.
swagger:response completedJobResponse
*/
// nolint
type CompletedJobResponse struct {
	// in: body
	Body bulkResponseBody
}

type bulkResponseBody struct {
	// Server time when the query was run
	TransactionTime time.Time `json:"transactionTime"`
	// URL of the bulk data export request
	RequestURL string `json:"request"`
	// Indicates whether an access token is required to download generated data files
	RequiresAccessToken bool `json:"requiresAccessToken"`
	// Information about generated data files, including URLs for downloading
	Files []fileItem `json:"output"`
	// Information about error files, including URLs for downloading
	Errors []fileItem `json:"error"`
	JobID  uint
}

func readAuthData(r *http.Request) (data auth.AuthData, err error) {
	var ok bool
	data, ok = r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
	if !ok {
		err = errors.New("no auth data in context")
	}
	return
}

func GetJobTimeout() time.Duration {
	return time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))
}

func SetQC(client *que.Client) {
	qc = client
}
