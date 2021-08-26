package api

import (
	"context"
	"database/sql"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"net/http"
	"time"

	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	responseutilsv2 "github.com/CMSgov/bcda-app/bcda/responseutils/v2"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
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

	supportedDataTypes map[string]service.DataType

	supportedResourceTypes []string

	bbBasePath string

	apiVersion string

	RespWriter fhirResponseWriter
}

type fhirResponseWriter interface {
	Exception(http.ResponseWriter, int, string, string)
	NotFound(http.ResponseWriter, int, string, string)
}

func NewHandler(dataTypes map[string]service.DataType, basePath string, apiVersion string) *Handler {
	return newHandler(dataTypes, basePath, apiVersion, database.Connection)
}

func newHandler(dataTypes map[string]service.DataType, basePath string, apiVersion string, db *sql.DB) *Handler {
	h := &Handler{JobTimeout: time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))}

	h.Enq = queueing.NewEnqueuer()

	cfg, err := service.LoadConfig()
	if err != nil {
		log.API.Fatalf("Failed to load service config. Err: %v", err)
	}

	repository := postgres.NewRepository(db)
	h.db, h.r = db, repository
	h.Svc = service.NewService(repository, cfg, basePath)

	h.supportedDataTypes = dataTypes

	// Build string array of supported Resource types
	h.supportedResourceTypes = make([]string, 0, len(h.supportedDataTypes))

	for k := range h.supportedDataTypes {
		h.supportedResourceTypes = append(h.supportedResourceTypes, k)
	}

	h.bbBasePath = basePath
	h.apiVersion = apiVersion

	switch h.apiVersion {
	case "v1":
		h.RespWriter = responseutils.NewResponseWriter()
	case "v2":
		h.RespWriter = responseutilsv2.NewResponseWriter()
	default:
		log.API.Fatalf("unexpected API version: %s", h.apiVersion)
	}

	return h
}

func (h *Handler) BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	reqType := service.DefaultRequest // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	if isALRRequest(r) {
		h.alrRequest(w, r, reqType)
		return
	}
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
		h.RespWriter.Exception(w, http.StatusBadRequest, responseutils.RequestErr, "Invalid group ID")
		return
	}

	if isALRRequest(r) {
		h.alrRequest(w, r, reqType)
		return
	}
	h.bulkRequest(w, r, reqType)
}

func (h *Handler) JobStatus(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		log.API.Error(err)
		h.RespWriter.Exception(w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return
	}

	job, jobKeys, err := h.Svc.GetJobAndKeys(context.Background(), uint(jobID))
	if err != nil {
		log.API.Error(err)
		// NOTE: This is a catch all and may not necessarily mean that the job was not found.
		// So returning a StatusNotFound may be a misnomer
		h.RespWriter.Exception(w, http.StatusNotFound, responseutils.DbErr, "")
		return
	}

	switch job.Status {

	case models.JobStatusFailed, models.JobStatusFailedExpired:
		h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "Service encountered numerous errors.  Unable to complete the request.")
	case models.JobStatusPending, models.JobStatusInProgress:
		w.Header().Set("X-Progress", job.StatusMessage())
		w.WriteHeader(http.StatusAccepted)
		return
	case models.JobStatusCompleted:
		// If the job should be expired, but the cleanup job hasn't run for some reason, still respond with 410
		if job.UpdatedAt.Add(h.JobTimeout).Before(time.Now()) {
			w.Header().Set("Expires", job.UpdatedAt.Add(h.JobTimeout).String())
			h.RespWriter.Exception(w, http.StatusGone, responseutils.NotFoundErr, "")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Expires", job.UpdatedAt.Add(h.JobTimeout).String())
		scheme := "http"
		if servicemux.IsHTTPS(r) {
			scheme = "https"
		}

		rb := BulkResponseBody{
			TransactionTime:     job.TransactionTime,
			RequestURL:          job.RequestURL,
			RequiresAccessToken: true,
			Files:               []FileItem{},
			Errors:              []FileItem{},
			JobID:               job.ID,
		}

		for _, jobKey := range jobKeys {
			// data files
			fi := FileItem{
				Type: jobKey.ResourceType,
				URL:  fmt.Sprintf("%s://%s/data/%d/%s", scheme, r.Host, jobID, strings.TrimSpace(jobKey.FileName)),
			}
			rb.Files = append(rb.Files, fi)

			// error files
			errFileName := strings.Split(jobKey.FileName, ".")[0]
			errFilePath := fmt.Sprintf("%s/%d/%s-error.ndjson", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID, errFileName)
			if _, err := os.Stat(errFilePath); !os.IsNotExist(err) {
				errFI := FileItem{
					Type: "OperationOutcome",
					URL:  fmt.Sprintf("%s://%s/data/%d/%s-error.ndjson", scheme, r.Host, jobID, errFileName),
				}
				rb.Errors = append(rb.Errors, errFI)
			}
		}

		jsonData, err := json.Marshal(rb)
		if err != nil {
			h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}

		w.WriteHeader(http.StatusOK)
	case models.JobStatusArchived, models.JobStatusExpired:
		w.Header().Set("Expires", job.UpdatedAt.Add(h.JobTimeout).String())
		h.RespWriter.Exception(w, http.StatusGone, responseutils.NotFoundErr, "")
	case models.JobStatusCancelled, models.JobStatusCancelledExpired:
		h.RespWriter.NotFound(w, http.StatusNotFound, responseutils.NotFoundErr, "Job has been cancelled.")

	}
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		log.API.Error(err)
		h.RespWriter.Exception(w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return
	}

	_, err = h.Svc.CancelJob(context.Background(), uint(jobID))
	if err != nil {
		switch err {
		case service.ErrJobNotCancellable:
			h.RespWriter.Exception(w, http.StatusGone, responseutils.DeletedErr, err.Error())
			return
		default:
			log.API.Error(err)
			h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.DbErr, err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

type AttributionFileStatus struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	CCLFNum   int       `json:"cclf_number"`
	Type      string    `json:"cclf_file_type"`
}

type AttributionFileStatusResponse struct {
	CCLFFiles []AttributionFileStatus `json:"cclf_files"`
}

func (h *Handler) AttributionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		ad   auth.AuthData
		err  error
		resp AttributionFileStatusResponse
	)

	if ad, err = readAuthData(r); err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Retrieve the most recent cclf 8 file we have successfully ingested
	asd, err := h.getAttributionFileStatus(ctx, ad.CMSID, models.FileTypeDefault)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the most recent cclf 8 runout file we have successfully ingested
	asr, err := h.getAttributionFileStatus(ctx, ad.CMSID, models.FileTypeRunout)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if asd == nil && asr == nil {
		h.RespWriter.Exception(w, http.StatusNotFound, responseutils.NotFoundErr, "")
		return
	}

	if asd != nil {
		resp.CCLFFiles = append(resp.CCLFFiles, *asd)
	}

	if asr != nil {
		resp.CCLFFiles = append(resp.CCLFFiles, *asr)
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getAttributionFileStatus(ctx context.Context, CMSID string, fileType models.CCLFFileType) (*AttributionFileStatus, error) {
	cclfFile, err := h.Svc.GetLatestCCLFFile(ctx, CMSID, fileType)
	if err != nil {
		log.API.Error(err)

		if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok {
			return nil, nil
		} else {
			return nil, err
		}
	}

	status := &AttributionFileStatus{
		Name:      cclfFile.Name,
		Timestamp: cclfFile.Timestamp,
		CCLFNum:   cclfFile.CCLFNum,
	}

	switch fileType {
	case models.FileTypeDefault:
		status.Type = "default"
	case models.FileTypeRunout:
		status.Type = "runout"
	}

	return status, nil
}

func (h *Handler) bulkRequest(w http.ResponseWriter, r *http.Request, reqType service.RequestType) {
	// Create context to encapsulate the entire workflow. In the future, we can define child context's for timing.
	ctx := r.Context()

	var (
		ad  auth.AuthData
		err error
	)

	if ad, err = readAuthData(r); err != nil {
		h.RespWriter.Exception(w, http.StatusUnauthorized, responseutils.TokenErr, "")
		return
	}

	rp, ok := middleware.RequestParametersFromContext(ctx)
	if !ok {
		panic("Request parameters must be set prior to calling this handler.")
	}

	resourceTypes := h.getResourceTypes(rp, ad.CMSID)

	if err = h.validateRequest(resourceTypes, ad.CMSID); err != nil {
		h.RespWriter.Exception(w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig(h.bbBasePath))
	if err != nil {
		log.API.Error(err)
		h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
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
		err = fmt.Errorf("failed to start transaction: %w", err)
		log.API.Error(err)
		h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
		return
	}
	// Use a transaction backed repository to ensure all of our upserts are encapsulated into a single transaction
	rtx := postgres.NewRepositoryTx(tx)

	defer func() {
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				log.API.Warnf("Failed to rollback transaction %s", err.Error())
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
			log.API.Error(err.Error())
			h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.DbErr, "")
			return
		}

		// We've successfully created the job
		w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/%s/jobs/%d", scheme, r.Host, h.apiVersion, newJob.ID))
		w.WriteHeader(http.StatusAccepted)
	}()

	newJob.ID, err = rtx.CreateJob(ctx, newJob)
	if err != nil {
		log.API.Error(err)
		h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.DbErr, "")
		return
	}

	// request a fake patient in order to acquire the bundle's lastUpdated metadata
	b, err := bb.GetPatient("FAKE_PATIENT", strconv.FormatUint(uint64(newJob.ID), 10), acoID.String(), "", time.Now())
	if err != nil {
		log.API.Error(err)
		h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.FormatErr, "Failure to retrieve transactionTime metadata from FHIR Data Server.")
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
		log.API.Error(err)
		var (
			respCode int
			errType  string
		)
		if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok && reqType == service.Runout {
			respCode = http.StatusNotFound
			errType = responseutils.NotFoundErr
		} else {
			respCode = http.StatusInternalServerError
			errType = responseutils.InternalErr
		}
		h.RespWriter.Exception(w, respCode, errType, err.Error())
		return
	}
	newJob.JobCount = len(queJobs)

	// We've now computed all of the fields necessary to populate a fully defined job
	if err = rtx.UpdateJob(ctx, newJob); err != nil {
		log.API.Error(err.Error())
		h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.DbErr, "")
		return
	}

	// Since we're enqueuing these queuejobs BEFORE we've created the actual job, we may encounter a transient
	// error where the job does not exist. Since queuejobs are retried, the transient error will be resolved
	// once we finish inserting the job.
	for _, j := range queJobs {
		sinceParam := (!rp.Since.IsZero() || conditions.ReqType == service.RetrieveNewBeneHistData)
		jobPriority := h.Svc.GetJobPriority(conditions.CMSID, j.ResourceType, sinceParam) // first argument is the CMS ID, not the ACO uuid

		if err = h.Enq.AddJob(*j, int(jobPriority)); err != nil {
			log.API.Error(err)
			h.RespWriter.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}
	}
}

func (h *Handler) getResourceTypes(parameters middleware.RequestParameters, cmsID string) []string {
	resourceTypes := parameters.ResourceTypes

	// If caller does not supply resource types, we default to all supported resource types for the specific ACO
	if len(resourceTypes) == 0 {
		if acoConfig, found := h.Svc.GetACOConfigForID(cmsID); found {
			if utils.ContainsString(acoConfig.Data, constants.Adjudicated) {
				resourceTypes = append(resourceTypes, "Patient", "ExplanationOfBenefit", "Coverage")
			}

			if utils.ContainsString(acoConfig.Data, constants.PreAdjudicated) {
				resourceTypes = append(resourceTypes, "Claim", "ClaimResponse")
			}
		}
	}

	return resourceTypes
}

func (h *Handler) validateRequest(resourceTypes []string, cmsID string) error {

	for _, resourceType := range resourceTypes {
		dataType, ok := h.supportedDataTypes[resourceType]

		if !ok {
			return fmt.Errorf("invalid resource type %s. Supported types %s", resourceType, h.supportedResourceTypes)
		}

		if !h.authorizedResourceAccess(dataType, cmsID) {
			return fmt.Errorf("unauthorized resource type %s", resourceType)
		}
	}

	return nil
}

func (h *Handler) authorizedResourceAccess(dataType service.DataType, cmsID string) bool {
	if cfg, ok := h.Svc.GetACOConfigForID(cmsID); ok {
		return (dataType.Adjudicated && utils.ContainsString(cfg.Data, constants.Adjudicated)) ||
			(dataType.PreAdjudicated && utils.ContainsString(cfg.Data, constants.PreAdjudicated))
	}

	return false
}

func readAuthData(r *http.Request) (data auth.AuthData, err error) {
	var ok bool
	data, ok = r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
	if !ok {
		err = goerrors.New("no auth data in context")
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
