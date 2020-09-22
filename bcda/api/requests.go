package api

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bgentry/que-go"
	fhirmodels "github.com/eug48/fhir/models"

	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

var (
	qc *que.Client
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

func BulkRequest(resourceTypes []string, w http.ResponseWriter, r *http.Request, retrieveNewBeneHistData bool) {
	var (
		ad  auth.AuthData
		err error
	)

	if ad, err = readAuthData(r); err != nil {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.TokenErr, "")
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	if qc == nil {
		err = errors.New("queue client not initialized")
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
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

	// Need to create job in transaction instead of the very end of the process because we need
	// the newJob.ID field to be set in the associated queuejobs. By doing the job creation (and update)
	// in a transaction, we can rollback if we encounter any errors with handling the data needed for the newJob
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			// We've already written out the HTTP response so we can return after we've rolled back the transaction
			return
		}

		// We create the job after populating all of the data needed for the job (including inserting all of the queue jobs) to
		// ensure that the job will be able to be processed and it WILL NOT BE stuck in the Pending state.
		// For example, we write that the job has 10 queuejobs. We fail after inserting 9 queuejobs. The job will
		// never move out of the IN_PROGRESS (or PENDING) state since we'll never be able to add the last queuejob.
		//
		// Since the queue jobs may (and do) exist in a different database, we cannot use a single transaction to encompass
		// both adding queuejobs and adding the parent job.
		//
		// This does introduce an error scenario where we have queuejobs but no parent job.
		// We've added logic into the worker to handle this situation.
		if err = tx.Commit().Error; err != nil {
			log.Error(err.Error())
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		// We've successfully create the job
		w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/v1/jobs/%d", scheme, r.Host, newJob.ID))
		w.WriteHeader(http.StatusAccepted)
	}()

	if err = tx.Save(&newJob).Error; err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// request a fake patient in order to acquire the bundle's lastUpdated metadata
	b, err := bb.GetPatient("FAKE_PATIENT", strconv.FormatUint(uint64(newJob.ID), 10), acoID, "", time.Now())
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "Failure to retrieve transactionTime metadata from FHIR Data Server.")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	newJob.TransactionTime = b.Meta.LastUpdated

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
	newJob.JobCount = len(enqueueJobs)

	// We've now computed all of the fields necessary to populate a fully defined job
	if err = tx.Save(&newJob).Error; err != nil {
		log.Error(err.Error())
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// Since we're enqueuing these queuejobs BEFORE we've created the actual job, we may encounter a transient
	// error where the job does not exist. Since queuejobs are retried, the transient error will be resolved
	// once we finish inserting the job.
	for _, j := range enqueueJobs {
		if err = qc.Enqueue(j); err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
	}
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

func ValidateRequest(r *http.Request) ([]string, *fhirmodels.OperationOutcome) {

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

// swagger:model fileItem
type FileItem struct {
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
	Body BulkResponseBody
}

type BulkResponseBody struct {
	// Server time when the query was run
	TransactionTime time.Time `json:"transactionTime"`
	// URL of the bulk data export request
	RequestURL string `json:"request"`
	// Indicates whether an access token is required to download generated data files
	RequiresAccessToken bool `json:"requiresAccessToken"`
	// Information about generated data files, including URLs for downloading
	Files []FileItem `json:"output"`
	// Information about error files, including URLs for downloading
	Errors []FileItem `json:"error"`
	JobID  uint
}
