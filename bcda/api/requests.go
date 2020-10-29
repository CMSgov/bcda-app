package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/jackc/pgx"
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

type Handler struct {
	qc  *que.Client
	svc models.Service

	supportedResources map[string]struct{}

	bbBasePath string
}

func NewHandler(resources []string, basePath string) *Handler {
	h := &Handler{}

	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		log.Fatal(err)
	}

	h.qc = que.NewClient(pgxpool)

	cutoffDuration := time.Duration(utils.GetEnvInt("CCLF_CUTOFF_DATE_DAYS", 45)*24) * time.Hour

	// Allow runout requests to be up to 4 months after runout data was ingested
	runoutCutoffDuration := time.Duration(utils.GetEnvInt("RUNOUT_CUTOFF_DATE_DAYS", 45)*24) * time.Hour
	db := database.GetGORMDbConnection()
	db.DB().SetMaxOpenConns(utils.GetEnvInt("BCDA_DB_MAX_OPEN_CONNS", 25))
	db.DB().SetMaxIdleConns(utils.GetEnvInt("BCDA_DB_MAX_IDLE_CONNS", 25))
	db.DB().SetConnMaxLifetime(time.Duration(utils.GetEnvInt("BCDA_DB_CONN_MAX_LIFETIME_MIN", 5)) * time.Minute)
	repository := postgres.NewRepository(db)
	h.svc = models.NewService(repository, cutoffDuration, utils.GetEnvInt("BCDA_SUPPRESSION_LOOKBACK_DAYS", 60), runoutCutoffDuration, basePath)

	h.supportedResources = make(map[string]struct{}, len(resources))
	for _, r := range resources {
		h.supportedResources[r] = struct{}{}
	}

	h.bbBasePath = basePath

	return h
}

func (h *Handler) BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	resourceTypes, err := h.validateRequest(r)
	if err != nil {
		responseutils.WriteError(err, w, http.StatusBadRequest)
		return
	}
	reqType := models.DefaultRequest // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	h.bulkRequest(resourceTypes, w, r, reqType)
}

func (h *Handler) BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	const (
		groupAll    = "all"
		groupRunout = "runout"
	)

	reqType := models.DefaultRequest
	groupID := chi.URLParam(r, "groupId")
	switch groupID {
	case groupAll:
		// Set flag to retrieve new beneficiaries' historical data if _since param is provided and feature is turned on
		_, ok := r.URL.Query()["_since"]
		if ok && utils.GetEnvBool("BCDA_ENABLE_NEW_GROUP", false) {
			reqType = models.RetrieveNewBeneHistData
		}
	case groupRunout:
		if utils.GetEnvBool("BCDA_ENABLE_RUNOUT", true) {
			reqType = models.Runout
			break
		}
		fallthrough
	default:
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Invalid group ID")
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	resourceTypes, err := h.validateRequest(r)
	if err != nil {
		responseutils.WriteError(err, w, http.StatusBadRequest)
		return
	}

	h.bulkRequest(resourceTypes, w, r, reqType)
}

func (h *Handler) bulkRequest(resourceTypes []string, w http.ResponseWriter, r *http.Request, reqType models.RequestType) {
	var (
		ad      auth.AuthData
		version string
		err     error
	)

	if version, err = getVersion(r.URL); err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, err.Error())
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
	}

	if ad, err = readAuthData(r); err != nil {
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.TokenErr, "")
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig(h.bbBasePath))
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	acoID := ad.ACOID

	var pendingAndInProgressJobs []models.Job
	// If we really do find this record with the below matching criteria then this particular ACO has already made
	// a bulk data request and it has yet to finish. Users will be presented with a 429 Too-Many-Requests error until either
	// their job finishes or time expires (+24 hours default) for any remaining jobs left in a pending or in-progress state.
	// Overall, this will prevent a queue of concurrent calls from slowing up our system.
	// NOTE: this logic is relevant to PROD only; simultaneous requests in our lower environments is acceptable (i.e., shared opensbx creds)
	if (os.Getenv("DEPLOYMENT_TARGET") == "prod") &&
		(!db.Find(&pendingAndInProgressJobs, "aco_id = ? AND status IN (?, ?)", acoID, "In Progress", "Pending").RecordNotFound()) {
		if types, err := check429(pendingAndInProgressJobs, resourceTypes, version); err != nil {
			if _, ok := err.(duplicateTypeError); ok {
				w.Header().Set("Retry-After", strconv.Itoa(utils.GetEnvInt("CLIENT_RETRY_AFTER_IN_SECONDS", 0)))
				w.WriteHeader(http.StatusTooManyRequests)
			} else {
				log.Error(err)
				oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
				responseutils.WriteError(oo, w, http.StatusInternalServerError)
			}

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

	var since time.Time
	// Decode the _since parameter (if it exists) so it can be persisted in job args
	if params, ok := r.URL.Query()["_since"]; ok {
		since, err = time.Parse(time.RFC3339Nano, params[0])
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
		}
	}

	var queJobs []*que.Job
	queJobs, err = h.svc.GetQueJobs(ad.CMSID, &newJob, resourceTypes, since, reqType)
	if err != nil {
		log.Error(err)
		respCode := http.StatusInternalServerError
		if _, ok := errors.Cause(err).(models.CCLFNotFoundError); ok && reqType == models.Runout {
			respCode = http.StatusNotFound
		}
		oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, err.Error())
		responseutils.WriteError(oo, w, respCode)
		return
	}
	newJob.JobCount = len(queJobs)

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
	for _, j := range queJobs {
		if err = h.qc.Enqueue(j); err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Processing, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) validateRequest(r *http.Request) ([]string, *fhirmodels.OperationOutcome) {

	// validate optional "_type" parameter
	var resourceTypes []string
	params, ok := r.URL.Query()["_type"]
	if ok {
		resourceMap := make(map[string]bool)
		params = strings.Split(params[0], ",")
		for _, p := range params {
			if !resourceMap[p] {
				resourceMap[p] = true
				resourceTypes = append(resourceTypes, p)
			} else {
				oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr, "Repeated resource type")
				return nil, oo
			}
		}
	} else {
		// resource types not supplied in request; default to applying all resource types.
		resourceTypes = append(resourceTypes, "Patient", "ExplanationOfBenefit", "Coverage")
	}

	for _, resourceType := range resourceTypes {
		if _, ok := h.supportedResources[resourceType]; !ok {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.RequestErr,
				fmt.Sprintf("Invalid resource type %s. Supported types %s.", resourceType, h.supportedResources))
			return nil, oo
		}
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

	// Check and see if the user has a duplicated the query parameter symbol (?)
	// e.g. /api/v1/Patient/$export?_type=ExplanationOfBenefit&?_since=2020-09-13T08:00:00.000-05:00
	for key := range r.URL.Query() {
		if strings.HasPrefix(key, "?") {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.FormatErr, "Invalid parameter: query parameters cannot start with ?")
			return nil, oo
		}
	}

	return resourceTypes, nil
}

type duplicateTypeError struct{}

func (e duplicateTypeError) Error() string {
	return "Duplicate type found"
}

// check429 verifies that we do not have a duplicate resource type request.
// Returns the unworkedTypes (if any)
func check429(jobs []models.Job, types []string, version string) ([]string, error) {
	var unworkedTypes []string

	for _, t := range types {
		worked := false
		for _, job := range jobs {
			req, err := url.Parse(job.RequestURL)
			if err != nil {
				return nil, err
			}
			jobVersion, err := getVersion(req)
			if err != nil {
				return nil, err
			}

			// We allow different API versions to trigger jobs with the same resource type
			if jobVersion != version {
				continue
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
					return nil, duplicateTypeError{}
				}
			}
		}
		if !worked {
			unworkedTypes = append(unworkedTypes, t)
		}
	}
	if len(unworkedTypes) == 0 {
		return nil, duplicateTypeError{}
	} else {
		return unworkedTypes, nil
	}
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

func getVersion(url *url.URL) (string, error) {
	re := regexp.MustCompile(`\/api\/(.*)\/[Patient|Group].*`)
	parts := re.FindStringSubmatch(url.Path)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected path provided %s", url.Path)
	}
	return parts[1], nil
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
