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
     - api_key:

     SecurityDefinitions:
     api_key:
          type: apiKey
          name: Authorization
          in: header
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
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/bgentry/que-go"
	"github.com/dgrijalva/jwt-go"
	fhirmodels "github.com/eug48/fhir/models"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

/*
  	swagger:route GET /api/v1/ExplanationOfBenefit/$export bulkData bulkRequest

	Start explanation of benefit export

	Initiates a job to collect data from the Blue Button API for your ACO.

	Produces:
	- application/fhir+json

	Schemes: [http, https]

	Security:
		api_key

	Responses:
		202:BulkRequestResponse
		400:ErrorModel
		500:FHIRResponse
*/
func bulkEOBRequest(w http.ResponseWriter, r *http.Request) {
	bulkRequest("ExplanationOfBenefit", w, r)
}

/*
	swagger:route GET /api/v1/Patient/$export bulkData bulkRequest

	Start patient data export

	Initiates a job to collect data from the Blue Button API for your ACO.

	Produces:
	- application/fhir+json

	Schemes: [http, https]

	Security:
		api_key

	Responses:
		202:BulkRequestResponse
		400:ErrorModel
		500:FHIRResponse
*/
func bulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	bulkRequest("Patient", w, r)
}

func bulkRequest(t string, w http.ResponseWriter, r *http.Request) {
	if t != "ExplanationOfBenefit" && t != "Patient" {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "Invalid resource type", responseutils.RequestErr)
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	m := monitoring.GetMonitor()
	txn := m.Start("bulkRequest", w, r)
	defer m.End(txn)

	var (
		claims jwt.MapClaims
		err    error
	)

	db := database.GetGORMDbConnection()
	defer db.Close()

	if claims, err = readTokenClaims(r); err != nil {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	acoId, _ := claims["aco"].(string)
	userId, _ := claims["sub"].(string)

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	newJob := models.Job{
		AcoID:      uuid.Parse(acoId),
		UserID:     uuid.Parse(userId),
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     "Pending",
	}
	if result := db.Save(&newJob); result.Error != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	beneficiaryIds := []string{}
	rows, err := db.Table("beneficiaries").Select("patient_id").Where("aco_id = ?", acoId).Rows()
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var id string
	for rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
		beneficiaryIds = append(beneficiaryIds, id)
	}

	// TODO(rnagle): this checks for ?encrypt=true appended to the bulk data request URL
	// This is a temporary addition to allow SCA/ACT auditors to verify encryption of files works properly
	// without exposing file encryption functionality to BCDA pilot users.
	var encrypt bool = false
	param, ok := r.URL.Query()["encrypt"]
	if ok && strings.ToLower(param[0]) == "true" {
		encrypt = true
	}

	args, err := json.Marshal(jobEnqueueArgs{
		ID:             int(newJob.ID),
		AcoID:          acoId,
		UserID:         userId,
		BeneficiaryIDs: beneficiaryIds,
		ResourceType:   t,
		// TODO(rnagle): remove `Encrypt` when file encryption functionality is ready for release
		Encrypt: encrypt,
	})
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	j := &que.Job{
		Type: "ProcessJob",
		Args: args,
	}

	if qc == nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	} else if err = qc.Enqueue(j); err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
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
		api_key:

	Responses:
		202:JobStatus
		200:bulkResponseBody
		400:ErrorModel
        404:ErrorModel
		500:FHIRResponse
*/
func jobStatus(w http.ResponseWriter, r *http.Request) {
	m := monitoring.GetMonitor()
	txn := m.Start("jobStatus", w, r)
	defer m.End(txn)

	jobID := chi.URLParam(r, "jobId")
	db := database.GetGORMDbConnection()
	defer db.Close()

	i, err := strconv.Atoi(jobID)
	if err != nil {
		log.Print(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	var claims jwt.MapClaims

	if claims, err = readTokenClaims(r); err != nil {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	acoId := claims["aco"].(string)

	var job models.Job
	err = db.Find(&job, "id = ? and aco_id = ?", i, acoId).Error
	if err != nil {
		log.Print(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusNotFound)
		return
	}

	switch job.Status {
	case "Pending":
		fallthrough
	case "In Progress":
		w.Header().Set("X-Progress", job.Status)
		w.WriteHeader(http.StatusAccepted)
	case "Failed":
		responseutils.WriteError(&fhirmodels.OperationOutcome{}, w, http.StatusInternalServerError)
	case "Completed":
		w.Header().Set("Content-Type", "application/json")

		scheme := "http"
		if servicemux.IsHTTPS(r) {
			scheme = "https"
		}

		re := regexp.MustCompile(`/(ExplanationOfBenefit|Patient)/\$export`)
		resourceType := re.FindStringSubmatch(job.RequestURL)[1]

		fi := fileItem{
			Type: resourceType,
			URL:  fmt.Sprintf("%s://%s/data/%s/%s.ndjson", scheme, r.Host, jobID, job.AcoID),
		}

		var jobKeys []string
		keyMap := make(map[string]string)
		var jobKeysObj []models.JobKey
		db.Find(&jobKeysObj, "job_id = ?", job.ID)
		for _, jobKey := range jobKeysObj {
			jobKeys = append(jobKeys, hex.EncodeToString(jobKey.EncryptedKey)+"|"+jobKey.FileName)
			keyMap[jobKey.FileName] = hex.EncodeToString(jobKey.EncryptedKey)
		}

		rb := bulkResponseBody{
			TransactionTime:     job.CreatedAt,
			RequestURL:          job.RequestURL,
			RequiresAccessToken: true,
			Files:               []fileItem{fi},
			Keys:                jobKeys,
			Errors:              []fileItem{},
			KeyMap:              keyMap,
		}

		errFilePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), jobID, job.AcoID)
		if _, err := os.Stat(errFilePath); !os.IsNotExist(err) {
			errFI := fileItem{
				Type: "OperationOutcome",
				URL:  fmt.Sprintf("%s://%s/data/%s/%s-error.ndjson", scheme, r.Host, jobID, job.AcoID),
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
		api_key:

	Responses:
		200:ExplanationOfBenefitNDJSON
		400:ErrorModel
        404:ErrorModel
		500:FHIRResponse
*/
func serveData(w http.ResponseWriter, r *http.Request) {
	m := monitoring.GetMonitor()
	txn := m.Start("serveData", w, r)
	defer m.End(txn)

	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	acoID := chi.URLParam(r, "acoID")
	jobID := chi.URLParam(r, "jobID")
	http.ServeFile(w, r, fmt.Sprintf("%s/%s/%s.ndjson", dataDir, jobID, acoID))
}

func getToken(w http.ResponseWriter, r *http.Request) {
	m := monitoring.GetMonitor()
	txn := m.Start("getToken", w, r)
	defer m.End(txn)

	authBackend := auth.InitAuthBackend()

	db := database.GetGORMDbConnection()
	defer db.Close()

	var user auth.User
	err := db.First(&user, "name = ?", "User One").Error
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	var aco auth.ACO
	err = db.First(&aco, "name = ?", "ACO Dev").Error
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// Generates a token for fake user and ACO combination
	token, err := authBackend.GenerateTokenString(user.UUID.String(), aco.UUID.String())
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	_, err = w.Write([]byte(token))
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

func readTokenClaims(r *http.Request) (jwt.MapClaims, error) {
	var (
		claims jwt.MapClaims
		err    error
	)

	t := r.Context().Value("token")
	if token, ok := t.(*jwt.Token); ok && token.Valid {
		claims, err = auth.ClaimsFromToken(token)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		err = errors.New("missing or invalid token")
		log.Error(err)
		return nil, err
	}

	return claims, nil
}
