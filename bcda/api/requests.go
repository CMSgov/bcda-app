package api

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"net/http"
	"time"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"

	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
)

type Handler struct {
	JobTimeout time.Duration

	Enq queueing.Enqueuer

	Svc service.Service

	// Needed to have access to the repository/db for lookup needed in the bulkRequest.
	// TODO (BCDA-3412): Remove this reference once we've captured all of the necessary
	// logic into a service method.
	r  models.Repository
	db *sql.DB

	supportedResources map[string]struct{}

	bbBasePath string
}

func NewHandler(resources []string, basePath string) *Handler {
	return newHandler(resources, basePath, database.Connection)
}

func newHandler(resources []string, basePath string, db *sql.DB) *Handler {
	h := &Handler{JobTimeout: time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))}

	h.Enq = queueing.NewEnqueuer()

	cfg, err := service.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load service config. Err: %v", err)
	}

	repository := postgres.NewRepository(db)
	h.db, h.r = db, repository
	h.Svc = service.NewService(repository, cfg, basePath)

	h.supportedResources = make(map[string]struct{}, len(resources))
	for _, r := range resources {
		h.supportedResources[r] = struct{}{}
	}

	h.bbBasePath = basePath

	return h
}

func (h *Handler) BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	reqType := service.DefaultRequest // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	h.bulkRequest(w, r, reqType)
}

func (h *Handler) BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	const (
		groupAll    = "all"
		groupRunout = "runout"
	)

	reqType := service.DefaultRequest
	groupID := chi.URLParam(r, "groupId")
	switch groupID {
	case groupAll:
		// Set flag to retrieve new beneficiaries' historical data if _since param is provided and feature is turned on

		_, ok := r.URL.Query()["_since"]
		if ok && utils.GetEnvBool("BCDA_ENABLE_NEW_GROUP", false) {
			reqType = service.RetrieveNewBeneHistData
		}
	case groupRunout:
		if utils.GetEnvBool("BCDA_ENABLE_RUNOUT", true) {
			reqType = service.Runout
			break
		}
		fallthrough
	default:
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "Invalid group ID")
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	h.bulkRequest(w, r, reqType)
}

func (h *Handler) bulkRequest(w http.ResponseWriter, r *http.Request, reqType service.RequestType) {
	// Create context to encapsulate the entire workflow. In the future, we can define child context's for timing.
	ctx := r.Context()

	var (
		ad  auth.AuthData
		err error
	)

	if ad, err = readAuthData(r); err != nil {
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.TokenErr, "")
		responseutils.WriteError(oo, w, http.StatusUnauthorized)
		return
	}

	rp, ok := middleware.RequestParametersFromContext(ctx)
	if !ok {
		panic("Request parameters must be set prior to calling this handler.")
	}

	resourceTypes := rp.ResourceTypes
	// If caller does not supply resource types, we default to all supported resource types
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"Patient", "ExplanationOfBenefit", "Coverage"}
	}

	if err = h.validateRequest(resourceTypes); err != nil {
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr,
			err.Error())
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig(h.bbBasePath))
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
			responseutils.InternalErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	acoID := uuid.Parse(ad.ACOID)

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	newJob := models.Job{
		ACOID:      acoID,
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     models.JobStatusPending,
	}

	// Need to create job in transaction instead of the very end of the process because we need
	// the newJob.ID field to be set in the associated queuejobs. By doing the job creation (and update)
	// in a transaction, we can rollback if we encounter any errors with handling the data needed for the newJob
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to start transaction")
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
			responseutils.InternalErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	// Use a transaction backed repository to ensure all of our upserts are encapsulated into a single transaction
	rtx := postgres.NewRepositoryTx(tx)

	defer func() {
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				log.Warnf("Failed to rollback transaction %s", err.Error())
			}
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
		if err = tx.Commit(); err != nil {
			log.Error(err.Error())
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.DbErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		// We've successfully created the job
		w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/v1/jobs/%d", scheme, r.Host, newJob.ID))
		w.WriteHeader(http.StatusAccepted)
	}()

	newJob.ID, err = rtx.CreateJob(ctx, newJob)
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// request a fake patient in order to acquire the bundle's lastUpdated metadata
	b, err := bb.GetPatient("FAKE_PATIENT", strconv.FormatUint(uint64(newJob.ID), 10), acoID.String(), "", time.Now())
	if err != nil {
		log.Error(err)
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.FormatErr, "Failure to retrieve transactionTime metadata from FHIR Data Server.")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}
	newJob.TransactionTime = b.Meta.LastUpdated

	var queJobs []*models.JobEnqueueArgs

	conditions := service.RequestConditions{
		ReqType:   reqType,
		Resources: resourceTypes,

		CMSID: ad.CMSID,
		ACOID: newJob.ACOID,

		JobID:           newJob.ID,
		Since:           rp.Since,
		TransactionTime: newJob.TransactionTime,
	}
	queJobs, err = h.Svc.GetQueJobs(ctx, conditions)
	if err != nil {
		log.Error(err)
		var (
			oo       *fhirmodels.OperationOutcome
			respCode int
		)
		if _, ok := errors.Cause(err).(service.CCLFNotFoundError); ok && reqType == service.Runout {
			oo = responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, err.Error())
			respCode = http.StatusNotFound
		} else {
			oo = responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, err.Error())
			respCode = http.StatusInternalServerError
		}
		responseutils.WriteError(oo, w, respCode)
		return
	}
	newJob.JobCount = len(queJobs)

	// We've now computed all of the fields necessary to populate a fully defined job
	if err = rtx.UpdateJob(ctx, newJob); err != nil {
		log.Error(err.Error())
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.DbErr, "")
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	// Since we're enqueuing these queuejobs BEFORE we've created the actual job, we may encounter a transient
	// error where the job does not exist. Since queuejobs are retried, the transient error will be resolved
	// once we finish inserting the job.
	for _, j := range queJobs {
		sinceParam := (!rp.Since.IsZero() || conditions.ReqType == service.RetrieveNewBeneHistData)
		jobPriority := h.Svc.GetJobPriority(conditions.CMSID, j.ResourceType, sinceParam) // first argument is the CMS ID, not the ACO uuid

		if err = h.Enq.AddJob(*j, int(jobPriority)); err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) validateRequest(resourceTypes []string) error {

	for _, resourceType := range resourceTypes {
		if _, ok := h.supportedResources[resourceType]; !ok {
			return fmt.Errorf("Invalid resource type %s. Supported types %s.", resourceType, h.supportedResources)
		}
	}

	return nil
}

func readAuthData(r *http.Request) (data auth.AuthData, err error) {
	var ok bool
	data, ok = r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
	if !ok {
		err = errors.New("no auth data in context")
	}
	return
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
