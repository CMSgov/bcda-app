package web

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"

	"github.com/bgentry/que-go"
	fhirmodels "github.com/eug48/fhir/models"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
)

var (
	qc      *que.Client
	version = "latest"
)

/*
  	swagger:route GET /api/v1/ExplanationOfBenefit/$export bulkData bulkEOBRequest

	Start explanation of benefit export

	Initiates a job to collect data from the Blue Button API for your ACO.

	Produces:
	- application/fhir+json

	Security:
		bearer_token:

	Responses:
		202: BulkRequestResponse
		400: badRequestResponse
		500: errorResponse
*/

func bulkEOBRequest(w http.ResponseWriter, r *http.Request) {
	bulkRequest("ExplanationOfBenefit", w, r)
}

/*
	swagger:route GET /api/v1/Patient/$export bulkData bulkPatientRequest

	Start patient data export

	Initiates a job to collect data from the Blue Button API for your ACO.

	Produces:
	- application/fhir+json

	Security:
		bearer_token:

	Responses:
		202: BulkRequestResponse
		400: badRequestResponse
		500: errorResponse
*/
func bulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	bulkRequest("Patient", w, r)
}

/*
	swagger:route GET /api/v1/Coverage/$export bulkData bulkCoverageRequest

	Start coverage data export

	Initiates a job to collect data from the Blue Button API for your ACO.

	Produces:
	- application/fhir+json

	Security:
		bearer_token:

	Responses:
		202: BulkRequestResponse
		400: badRequestResponse
		500: errorResponse
*/
func bulkCoverageRequest(w http.ResponseWriter, r *http.Request) {
	bulkRequest("Coverage", w, r)
}

func bulkRequest(t string, w http.ResponseWriter, r *http.Request) {
	if t != "ExplanationOfBenefit" && t != "Patient" && t != "Coverage" {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "Invalid resource type", responseutils.RequestErr)
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	var (
		ad  auth.AuthData
		err error
	)

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	if ad, err = readAuthData(r); err != nil {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	acoID := ad.ACOID
	user := models.User{}
	// Arbitrarily use the first user in order to satisfy foreign key constraint "jobs_user_id_fkey" until user is removed from jobs table
	db.First(&user)
	userID := user.UUID

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	newJob := models.Job{
		ACOID:      uuid.Parse(acoID),
		UserID:     userID,
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     "Pending",
	}
	if result := db.Save(&newJob); result.Error != nil {
		log.Error(result.Error.Error())
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	if qc == nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	enqueueJobs, err := newJob.GetEnqueJobs(t)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	for _, j := range enqueueJobs {
		if err = qc.Enqueue(j); err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
	}

	if db.Model(&newJob).Update("job_count", len(enqueueJobs)).Error != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/v1/jobs/%d", scheme, r.Host, newJob.ID))
	w.WriteHeader(http.StatusAccepted)
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
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusNotFound)
		return
	}

	switch job.Status {

	case "Failed":
		responseutils.WriteError(&fhirmodels.OperationOutcome{}, w, http.StatusInternalServerError)
	case "Pending":
		fallthrough
	case "In Progress":
		// Check the job status in case it is done and just needs a small poke
		complete, err := job.CheckCompletedAndCleanup()

		if err != nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
		if !complete {
			w.Header().Set("X-Progress", job.Status)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		fallthrough

	case "Completed":
		// If the job should be expired, but the cleanup job hasn't run for some reason, still respond with 410
		if job.CreatedAt.Add(GetJobTimeout()).Before(time.Now()) {
			w.Header().Set("Expires", job.CreatedAt.Add(GetJobTimeout()).String())
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Deleted)
			responseutils.WriteError(oo, w, http.StatusGone)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Expires", job.CreatedAt.Add(GetJobTimeout()).String())
		scheme := "http"
		if servicemux.IsHTTPS(r) {
			scheme = "https"
		}

		re := regexp.MustCompile(`/(ExplanationOfBenefit|Patient|Coverage)/\$export`)
		resourceType := re.FindStringSubmatch(job.RequestURL)[1]

		var files []fileItem
		keyMap := make(map[string]string)
		var jobKeysObj []models.JobKey
		db.Find(&jobKeysObj, "job_id = ?", job.ID)
		for _, jobKey := range jobKeysObj {
			keyMap[strings.TrimSpace(jobKey.FileName)] = hex.EncodeToString(jobKey.EncryptedKey)
			fi := fileItem{
				Type:         resourceType,
				URL:          fmt.Sprintf("%s://%s/data/%s/%s", scheme, r.Host, jobID, strings.TrimSpace(jobKey.FileName)),
				EncryptedKey: hex.EncodeToString(jobKey.EncryptedKey),
			}
			files = append(files, fi)
		}

		rb := bulkResponseBody{
			TransactionTime:     job.CreatedAt,
			RequestURL:          job.RequestURL,
			RequiresAccessToken: true,
			Files:               files,
			Errors:              []fileItem{},
			KeyMap:              keyMap,
			JobID:               job.ID,
		}

		errFilePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), jobID, job.ACOID)
		if _, err := os.Stat(errFilePath); !os.IsNotExist(err) {
			errFI := fileItem{
				Type: "OperationOutcome",
				URL:  fmt.Sprintf("%s://%s/data/%s/%s-error.ndjson", scheme, r.Host, jobID, job.ACOID),
			}
			rb.Errors = append(rb.Errors, errFI)
		}

		jsonData, err := json.Marshal(rb)
		if err != nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	case "Archived":
		fallthrough
	case "Expired":
		w.Header().Set("Expires", job.CreatedAt.Add(GetJobTimeout()).String())
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Deleted)
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
		200: ExplanationOfBenefitNDJSON
		400: badRequestResponse
        404: notFoundResponse
		500: errorResponse
*/
func serveData(w http.ResponseWriter, r *http.Request) {
	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	fileName := chi.URLParam(r, "fileName")
	jobID := chi.URLParam(r, "jobID")
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
	statement := responseutils.CreateCapabilityStatement(dt, version, host)
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
	respMap["version"] = version
	respBytes, err := json.Marshal(respMap)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.InternalErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respBytes)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.InternalErr)
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
	// Encrypted Symmetric Key used to encrypt this file
	EncryptedKey string `json:"encryptedKey"`
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
	Errors []fileItem        `json:"error"`
	KeyMap map[string]string `json:"KeyMap"`
	JobID  uint
}

func readAuthData(r *http.Request) (data auth.AuthData, err error) {
	var ok bool
	data, ok = r.Context().Value("ad").(auth.AuthData)
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
