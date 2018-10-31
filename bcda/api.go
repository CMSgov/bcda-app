/*
 Package main BCDA API.

 The purpose of this application is to provide an application that allows for downloading of Beneficiary claims

 Terms Of Service:

 there are no TOS at this moment, use at your own risk we take no responsibility

	Schemes: http, https
     Host: localhost
     BasePath: /v2
     Version: 1.0.0
     License: https://github.com/CMSgov/bcda-app/blob/master/LICENSE.md
     Contact: bcapi@cms.hhs.gov

     Consumes:
     - application/json
     - application/xml

     Produces:
     - application/json
     - application/xml

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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/bgentry/que-go"
	"github.com/dgrijalva/jwt-go"
	fhirmodels "github.com/eug48/fhir/models"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

/*
	swagger:route GET  /api/v1/ExplanationOfBenefit/$export bulkData bulkRequest
	bulkRequest initiates a job to collect data from the Blue Button API for your ACO
	Consumes:
	- application/JSON
	Produces:
	- application/JSON
	Schemes: http, https
	Security:
		api_key
	Responses:
		default: BulkRequestResponse
		202:BulkRequestResponse
		400:ErrorModel
		500:FHIRResponse
*/
func bulkRequest(w http.ResponseWriter, r *http.Request) {
	m := monitoring.GetMonitor()
	txn := m.Start("bulkRequest", w, r)
	defer m.End(txn)

	var (
		claims jwt.MapClaims
		err    error
	)

	db := database.GetGORMDbConnection()
	defer db.Close()

	t := r.Context().Value("token")
	if token, ok := t.(*jwt.Token); ok && token.Valid {
		claims, err = auth.ClaimsFromToken(token)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
			responseutils.WriteError(oo, w, http.StatusUnauthorized)
			return
		}
	} else {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	acoId, _ := claims["aco"].(string)
	userId, _ := claims["sub"].(string)

	scheme := "http"
	if r.TLS != nil {
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

	args, err := json.Marshal(jobEnqueueArgs{
		ID:             int(newJob.ID),
		AcoID:          acoId,
		UserID:         userId,
		BeneficiaryIDs: beneficiaryIds,
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
	swagger:route GET /api/v1/jobs/{jobid} bulkData jobStatus
	jobStatus is the current status of a requested job.
	Consumes:
	- application/JSON
	Produces:
	- application/JSON
	Schemes: http, https
	Security:
		api_key:
	Responses:
		default: bulkResponseBody
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
	var job models.Job
	err = db.First(&job, i).Error
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
		if r.TLS != nil {
			scheme = "https"
		}

		fi := fileItem{
			Type: "ExplanationOfBenefit",
			URL:  fmt.Sprintf("%s://%s/data/%s/%s.ndjson", scheme, r.Host, jobID, job.AcoID),
		}

		var jobKeys []string
		var keyMap map[string]string
		for _, jobKey := range job.JobKeys {
			jobKeys = append(jobKeys, jobKey.EncryptedKey+"|"+jobKey.FileName)
			keyMap[jobKey.EncryptedKey] = jobKey.FileName
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

	// Generates a token for fake user and ACO combination
	token, err := authBackend.GenerateTokenString(
		"82503A18-BF3B-436D-BA7B-BAE09B7FFD2F",
		"3461c774-b48f-11e8-96f8-529269fb1459",
	)
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

func blueButtonMetadata(w http.ResponseWriter, r *http.Request) {
	bbClient, err := client.NewBlueButtonClient()
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	bbData, err := bbClient.GetMetadata()
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(bbData))
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func metadata(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := fmt.Sprintf("%s://%s", scheme, r.Host)
	statement := responseutils.CreateCapabilityStatement(dt, version, host)
	responseutils.WriteCapabilityStatement(statement, w)
}

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
