package web

import (
	"encoding/json"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/models/postgres"

	"net/http"
	"os"
	"strings"
	"time"

	fhirmodels "github.com/eug48/fhir/models"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

// var (
// 	qc *que.Client
// )

// const (
// 	groupAll = "all"
// )

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
	swagger:route GET /api/v2/Patient/$export bulkData bulkPatientRequest

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
func bulkPatientRequestV2(w http.ResponseWriter, r *http.Request) {
	resourceTypes, err := validateRequest(r)
	if err != nil {
		responseutils.WriteError(err, w, http.StatusBadRequest)
		return
	}
	retrieveNewBeneHistData := false // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	bulkRequest(resourceTypes, w, r, retrieveNewBeneHistData)
}

/*
	swagger:route GET /api/v2/Group/{groupId}/$export bulkData bulkGroupRequest

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
func bulkGroupRequestV2(w http.ResponseWriter, r *http.Request) {
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

/*
	swagger:route GET /api/v2/jobs/{jobId} bulkData jobStatus

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
func jobStatusV2(w http.ResponseWriter, r *http.Request) {
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
