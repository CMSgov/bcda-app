/*
 Package main Beneficiary Claims Data API

 The Beneficiary Claims Data API (BCDA) allows downloading of claims data in accordance with the FHIR Bulk Data Export specification.

 If you have a token you can use this page to explore the API.  To do this click the green "Authorize" button below and enter "Bearer {YOUR_TOKEN}"
 in the "Value" field and click authorize.  Until you click logout your token will be presented with every request made.  To make requests click on the
 "Try it out" button for the desired endpoint.


     Version: 1.0.0
     License: https://github.com/CMSgov/bcda-app/blob/master/LICENSE.md
     Contact: bcapi@cms.hhs.gov

     Produces:
     - application/fhir+json
     - application/json

     Security:
     - bearer_token: []

     SecurityDefinitions:
     bearer_token:
          type: apiKey
          name: For bulkData endpoints. 1) Put your credentials in Basic Authentication, 2) get your token from /auth/token, 3) put your token here, after the word "Bearer"
          in: header
     basic_auth:
          type: basic

 swagger:meta
*/
package main

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

/*
  	swagger:route GET /api/v1/ExplanationOfBenefit/$export bulkData bulkEOBRequest

	Start explanation of benefit export

	Initiates a job to collect data from the Blue Button API for your ACO.

	Produces:
	- application/fhir+json

	Security:
		bearer_token

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
		bearer_token

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
		bearer_token

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
	userID := ad.UserID

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	newJob := models.Job{
		ACOID:      uuid.Parse(acoID),
		UserID:     uuid.Parse(userID),
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     "Pending",
	}
	if result := db.Save(&newJob); result.Error != nil {
		log.Error(result.Error.Error())
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// TODO: this checks for ?encrypt=false appended to the bulk data request URL
	// By default, our encryption process is enabled but for now we are giving users the ability to turn
	// it off
	// Eventually, we will remove the ability for users to turn it off and it will remain on always
	var encrypt = true
	param, ok := r.URL.Query()["encrypt"]
	if ok && strings.ToLower(param[0]) == "false" {
		encrypt = false
	}

	if qc == nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	enqueueJobs, err := newJob.GetEnqueJobs(encrypt, t)
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
	swagger:route POST /auth/token auth getAuthToken

	Get access token

	Verifies Basic authentication credentials, and returns a JWT bearer token that can be presented to the other API endpoints.

	Produces:
	- application/json

	Schemes: https

	Security:
		basic_auth:

	Responses:
		200: tokenResponse
		400: missingCredentials
		401: invalidCredentials
		500: serverError
*/
func getAuthToken(w http.ResponseWriter, r *http.Request) {
	clientId, secret, ok := r.BasicAuth()
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	token, err := auth.GetProvider().MakeAccessToken(auth.Credentials{ClientID: clientId, ClientSecret: secret})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1
	// not included: recommended field expires_in
	body := []byte(fmt.Sprintf(`{"bearer_token": "%s","token_type":"bearer"}`, token))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	log.WithField("client_id", clientId).Println("issued access token")
}

func getToken(w http.ResponseWriter, r *http.Request) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var aco models.ACO
	err := db.First(&aco, "name = ?", "ACO Dev").Error
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// Generates a token for 'ACO Dev' and its first user
	token, err := auth.GetProvider().RequestAccessToken(auth.Credentials{ClientID: aco.UUID.String()}, 72)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(token.TokenString))
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
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
