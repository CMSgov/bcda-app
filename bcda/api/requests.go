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

	"github.com/go-chi/chi/v5"

	"github.com/pkg/errors"

	"net/http"
	"time"

	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
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
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	m "github.com/CMSgov/bcda-app/middleware"
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

	supportedStatuses map[models.JobStatus]struct{}

	bbBasePath string

	apiVersion string

	RespWriter fhirResponseWriter

	cfg *service.Config
}

// type RequestConditions struct {
// 	// ReqType    DataRequestType
// 	Resources  []string
// 	BBBasePath string

// 	CMSID string
// 	ACOID uuid.UUID

// 	JobID           uint
// 	Since           time.Time
// 	TransactionTime time.Time
// 	CreationTime    time.Time

// 	// Fields set in the service
// 	fileType models.CCLFFileType

// 	timeConstraint
// }

type fhirResponseWriter interface {
	Exception(context.Context, http.ResponseWriter, int, string, string)
	NotFound(context.Context, http.ResponseWriter, int, string, string)
	JobsBundle(context.Context, http.ResponseWriter, []*models.Job, string)
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
	h.cfg = cfg

	repository := postgres.NewRepository(db)
	h.db, h.r = db, repository
	h.Svc = service.NewService(repository, cfg, basePath)

	h.supportedDataTypes = dataTypes

	// Build string array of supported Resource types
	h.supportedResourceTypes = make([]string, 0, len(h.supportedDataTypes))

	for k := range h.supportedDataTypes {
		h.supportedResourceTypes = append(h.supportedResourceTypes, k)
	}

	h.supportedStatuses = make(map[models.JobStatus]struct{}, len(models.AllJobStatuses))
	for _, r := range models.AllJobStatuses {
		h.supportedStatuses[r] = struct{}{}
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
	reqType := constants.DefaultRequest // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	h.bulkRequest(w, r, reqType)
}

func (h *Handler) ALRRequest(w http.ResponseWriter, r *http.Request) {
	h.alrRequest(w, r)
}

func (h *Handler) BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	const (
		groupAll    = "all"
		groupRunout = "runout"
	)

	reqType := constants.DefaultRequest
	groupID := chi.URLParam(r, "groupId")
	switch groupID {
	case groupAll:
		// Set flag to retrieve new beneficiaries' historical data if _since param is provided and feature is turned on

		_, ok := r.URL.Query()["_since"]
		if ok && utils.GetEnvBool("BCDA_ENABLE_NEW_GROUP", false) {
			reqType = constants.RetrieveNewBeneHistData
		}
	case groupRunout:
		if utils.GetEnvBool("BCDA_ENABLE_RUNOUT", true) {
			reqType = constants.Runout
			break
		}
		fallthrough
	default:
		h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, "Invalid group ID")
		return
	}
	h.bulkRequest(w, r, reqType)
}

func (h *Handler) JobsStatus(w http.ResponseWriter, r *http.Request) {
	var (
		ad          auth.AuthData
		statusTypes []models.JobStatus
		err         error
	)
	logger := log.GetCtxLogger(r.Context())
	statusTypes = models.AllJobStatuses // default request to retrieve jobs with all statuses
	params, ok := r.URL.Query()["_status"]
	if ok {
		statusMap := make(map[string]struct{})
		rawStatusTypes := strings.Split(params[0], ",")
		statusTypes = nil

		// validate no duplicate status types
		for _, status := range rawStatusTypes {
			if _, ok := statusMap[status]; !ok {
				statusMap[status] = struct{}{}
				statusTypes = append(statusTypes, models.JobStatus(status))
			} else {
				errMsg := fmt.Sprintf("Repeated status type %s", status)
				h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
				return
			}
		}

		// validate status types provided match our valid list of statuses
		if err = h.validateStatuses(statusTypes); err != nil {
			logger.Error(err)
			h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
			return
		}
	}

	if ad, err = GetAuthDataFromCtx(r); err != nil {
		logger.Error(err)
		h.RespWriter.Exception(r.Context(), w, http.StatusUnauthorized, responseutils.TokenErr, "")
		return
	}

	jobs, err := h.Svc.GetJobs(r.Context(), uuid.Parse(ad.ACOID), statusTypes...)
	if err != nil {
		logger.Error(err)

		if ok := goerrors.As(err, &service.JobsNotFoundError{}); ok {
			h.RespWriter.Exception(r.Context(), w, http.StatusNotFound, responseutils.DbErr, err.Error())
		} else {
			h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
		}
	}

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	host := fmt.Sprintf("%s://%s", scheme, r.Host)

	// pass in the ctx here and log with the ctx logger
	h.RespWriter.JobsBundle(r.Context(), w, jobs, host)
}

func (h *Handler) validateStatuses(statusTypes []models.JobStatus) error {
	for _, statusType := range statusTypes {
		if _, ok := h.supportedStatuses[statusType]; !ok {
			return fmt.Errorf("invalid status type %s. Supported types %s", statusType, h.supportedStatuses)
		}
	}

	return nil
}

func (h *Handler) JobStatus(w http.ResponseWriter, r *http.Request) {
	logger := log.GetCtxLogger(r.Context())
	jobIDStr := chi.URLParam(r, "jobID")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		logger.Error(err)
		//We don't need to return the full error to a consumer.
		//We pass a bad request header (400) for this exception due to the inputs always being invalid for our purposes
		h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, "could not parse job id")

		return
	}

	job, jobKeys, err := h.Svc.GetJobAndKeys(r.Context(), uint(jobID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.API.Info("Requested job not found.", err.Error())
			h.RespWriter.Exception(r.Context(), w, http.StatusNotFound, responseutils.DbErr, "Job not found.")
		} else {
			log.API.Error("Error attempting to request job. Error:", err.Error())
			h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "Error trying to fetch job. Please try again.")
		}

		return
	}

	switch job.Status {

	case models.JobStatusFailed, models.JobStatusFailedExpired:
		logger.Error(job.Status)
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.JobFailed, responseutils.DetailJobFailed)
	case models.JobStatusPending, models.JobStatusInProgress:
		completedJobKeyCount := utils.CountUniq(jobKeys, func(jobKey *models.JobKey) int64 {
			if jobKey.QueJobID == nil {
				return -1
			}
			return *jobKey.QueJobID
		})
		w.Header().Set("X-Progress", job.StatusMessage(completedJobKeyCount))
		w.WriteHeader(http.StatusAccepted)
		return
	case models.JobStatusCompleted:
		// If the job should be expired, but the cleanup job hasn't run for some reason, still respond with 410
		if job.UpdatedAt.Add(h.JobTimeout).Before(time.Now()) {
			w.Header().Set("Expires", job.UpdatedAt.Add(h.JobTimeout).String())
			h.RespWriter.Exception(r.Context(), w, http.StatusGone, responseutils.NotFoundErr, "")
			return
		}
		w.Header().Set("Content-Type", constants.JsonContentType)
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

			// Check if "error" is not in the filename
			if !strings.Contains(strings.ToLower(jobKey.FileName), "-error.ndjson") {
				rb.Files = append(rb.Files, fi)
			}

			// Error files
			errFileName := strings.Split(jobKey.FileName, ".")[0]
			errFilePath := fmt.Sprintf("%s/%d/%s-error.ndjson", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID, errFileName)

			// Check if the error file exists
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
			logger.Error(err)
			h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			logger.Error(err)
			h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}

		w.WriteHeader(http.StatusOK)
	case models.JobStatusArchived, models.JobStatusExpired:
		w.Header().Set("Expires", job.UpdatedAt.Add(h.JobTimeout).String())
		h.RespWriter.Exception(r.Context(), w, http.StatusGone, responseutils.NotFoundErr, "")
	case models.JobStatusCancelled, models.JobStatusCancelledExpired:
		h.RespWriter.NotFound(r.Context(), w, http.StatusNotFound, responseutils.NotFoundErr, "Job has been cancelled.")

	}
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	logger := log.GetCtxLogger(r.Context())
	jobIDStr := chi.URLParam(r, "jobID")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		logger.Error(err)
		h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return
	}

	_, err = h.Svc.CancelJob(r.Context(), uint(jobID))
	if err != nil {
		switch err {
		case service.ErrJobNotCancellable:
			logger.Info(errors.Wrap(err, "Job is not cancellable"))
			h.RespWriter.Exception(r.Context(), w, http.StatusGone, responseutils.DeletedErr, err.Error())
			return
		default:
			logger.Error(err)
			h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.DbErr, err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

type AttributionFileStatus struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

type AttributionFileStatusResponse struct {
	Data []AttributionFileStatus `json:"ingestion_dates"`
}

func (h *Handler) AttributionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.GetCtxLogger(ctx)

	var (
		ad   auth.AuthData
		err  error
		resp AttributionFileStatusResponse
	)

	if ad, err = GetAuthDataFromCtx(r); err != nil {
		logger.Error(err)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Retrieve the most recent cclf 8 file we have successfully ingested
	group := chi.URLParam(r, "groupId")
	asd, err := h.getAttributionFileStatus(ctx, ad.CMSID, models.FileTypeDefault)
	if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok {
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.NotFoundErr, fmt.Sprintf("Unable to perform export operations for this Group. No up-to-date attribution information is available for Group '%s'. Usually this is due to awaiting new attribution information at the beginning of a Performance Year.", group))
		return
	}
	if asd != nil {
		resp.Data = append(resp.Data, *asd)
	}

	// Retrieve the most recent cclf 8 runout file we have successfully ingested
	asr, err := h.getAttributionFileStatus(ctx, ad.CMSID, models.FileTypeRunout)
	if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok {
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.NotFoundErr, fmt.Sprintf("Unable to perform export operations for this Group. No up-to-date attribution information is available for Group '%s'. Usually this is due to awaiting new attribution information at the beginning of a Performance Year.", group))
		return
	}
	if asr != nil {
		resp.Data = append(resp.Data, *asr)
	}

	if resp.Data == nil {
		logger.Error(errors.New("Could not find any CCLF8 files"))
		h.RespWriter.Exception(r.Context(), w, http.StatusNotFound, responseutils.NotFoundErr, "")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error(errors.Wrap(err, "Failed to encode JSON response"))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getAttributionFileStatus(ctx context.Context, CMSID string, fileType models.CCLFFileType) (*AttributionFileStatus, error) {
	logger := log.GetCtxLogger(ctx)
	cclfFile, err := h.Svc.GetLatestCCLFFile(ctx, CMSID, time.Time{}, time.Time{}, fileType)
	if err != nil {
		logger.Error(err)

		if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok {
			return nil, nil
		} else {
			return nil, err
		}
	}

	status := &AttributionFileStatus{
		Timestamp: cclfFile.Timestamp,
	}

	switch fileType {
	case models.FileTypeDefault:
		status.Type = "last_attribution_update"
	case models.FileTypeRunout:
		status.Type = "last_runout_update"
	}

	return status, nil
}

// bulkRequest generates a job ID for a bulk export request. It will not queue a job
// until auth, attribution, and request resources are validated.
func (h *Handler) bulkRequest(w http.ResponseWriter, r *http.Request, reqType constants.DataRequestType) {
	fmt.Printf("\n-------- Start bulkRequest, reqType: %+v\n", reqType)
	// Create context to encapsulate the entire workflow. In the future, we can define child context's for timing.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	logger := log.GetCtxLogger(ctx)

	var (
		ad  auth.AuthData
		err error
	)

	fmt.Println("\n--- start GetAuthDataFromCtx")
	if ad, err = GetAuthDataFromCtx(r); err != nil {
		logger.Error(err)
		h.RespWriter.Exception(r.Context(), w, http.StatusUnauthorized, responseutils.TokenErr, "")
		return
	}
	fmt.Printf("\n--- ad %+v\n", ad)

	fmt.Println("\n--- start GetACOConfigForID")
	if acoCfg, ok := h.Svc.GetACOConfigForID(ad.CMSID); ok {
		ctx = service.NewACOCfgCtx(ctx, acoCfg)
		fmt.Printf("\n--- acoCfg %+v\n", acoCfg)
	}

	fmt.Println("\n--- start GetRequestParamsFromCtx")
	rp, ok := middleware.GetRequestParamsFromCtx(ctx)
	if !ok {
		panic("Request parameters must be set prior to calling this handler.")
	}
	fmt.Printf("\n--- rp %+v\n", rp)

	fmt.Println("\n--- start getResourceTypes")
	resourceTypes := h.getResourceTypes(rp, ad.CMSID)
	fmt.Printf("\n--- resourceTypes %+v\n", resourceTypes)

	fmt.Println("\n--- start validateResources")
	if err = h.validateResources(resourceTypes, ad.CMSID); err != nil {
		logger.Error(err)
		h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return
	}

	fmt.Println("\n--- start job stuff")
	acoID := uuid.Parse(ad.ACOID)
	fmt.Printf("\n--- acoId: %+v\n", acoID)

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	newJob := models.Job{
		ACOID:      acoID,
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     models.JobStatusPending,
	}

	// tx, err := h.db.BeginTx(ctx, nil)
	// if err != nil {
	// 	err = fmt.Errorf("failed to start transaction: %w", err)
	// 	logger.Error(err)
	// 	h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
	// 	return
	// }
	// // Use a transaction backed repository to ensure all of our upserts are encapsulated into a single transaction
	// rtx := postgres.NewRepositoryTx(tx)

	// defer func() {
	// 	if err != nil {
	// 		if err1 := tx.Rollback(); err1 != nil {
	// 			logger.Warnf("Failed to rollback transaction %s", err.Error())
	// 		}
	// 		// We've already written out the HTTP response so we can return after we've rolled back the transaction
	// 		return
	// 	}

	// 	if err = tx.Commit(); err != nil {
	// 		logger.Error(err.Error())
	// 		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.DbErr, "")
	// 		return
	// 	}

	// 	// We've successfully created the job
	// 	w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/%s/jobs/%d", scheme, r.Host, h.apiVersion, newJob.ID))
	// 	w.WriteHeader(http.StatusAccepted)
	// }()

	// repo := postgres.NewRepository(h.db)

	// ------------------------------------------------------------------------------------------------

	// set before quejobs
	// conditions := RequestConditions{
	// 	// ReqType:    reqType,
	// 	Resources:  resourceTypes,
	// 	BBBasePath: h.bbBasePath,

	// 	// CMSID: ad.CMSID,
	// 	// ACOID: acoID,

	// 	// JobID:           args.Job.ID,
	// 	// Since: rp.Since,
	// 	// TransactionTime: args.Job.TransactionTime,
	// 	// CreationTime: time.Now(),

	// 	// set DEFAULT OR NEW AND EXISTING
	// }

	// quejobs logic
	// are timeconstraints used anywhere?
	var cutoffTime time.Time
	var timeConstraints service.TimeConstraints
	var cclfFileOldID uint
	var complexDataRequestType string
	var fileType models.CCLFFileType

	fmt.Println("\n--- Start setTimeConstraints")
	timeConstraints, err = h.Svc.SetTimeConstraints(ctx, ad.CMSID)
	if err != nil {
		fmt.Printf("\n--- time constraints err: %+v", err)
		logger.Error(err)
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
		return
	}
	fmt.Printf("\n--- setTimeConstraints: %+v", timeConstraints)

	hasAttributionDate := !timeConstraints.AttributionDate.IsZero()
	fmt.Printf("\n--- hasAttributionDate: %+v", hasAttributionDate)
	if reqType == constants.Runout {
		fileType = models.FileTypeRunout
	} else {
		fileType = models.FileTypeDefault
	}
	fmt.Printf("\n--- fileType: %+v", fileType)
	// for default requests, runouts, or any requests where the Since parameter is
	// after a terminated ACO's attribution date, we should only retrieve exisiting benes
	if reqType == constants.DefaultRequest ||
		reqType == constants.Runout ||
		(hasAttributionDate && rp.Since.After(timeConstraints.AttributionDate)) {
		complexDataRequestType = constants.GetExistingBenes
		// only set a cutoffTime if there is no attributionDate set
		if timeConstraints.AttributionDate.IsZero() {
			if fileType == models.FileTypeDefault && h.cfg.CutoffDuration > 0 {
				cutoffTime = time.Now().Add(-1 * h.cfg.CutoffDuration)
			} else if fileType == models.FileTypeRunout && h.cfg.RunoutConfig.CutoffDuration > 0 {
				cutoffTime = time.Now().Add(-1 * h.cfg.RunoutConfig.CutoffDuration)
			}
		}
	} else if reqType == constants.RetrieveNewBeneHistData {
		complexDataRequestType = constants.GetNewAndExistingBenes
		// only set cutoffTime if there is no attributionDate set
		if h.cfg.CutoffDuration > 0 && timeConstraints.AttributionDate.IsZero() {
			cutoffTime = time.Now().Add(-1 * h.cfg.CutoffDuration)
		}
	} else {
		logger.Error("invalid complex data request type")
		h.RespWriter.Exception(r.Context(), w, http.StatusBadRequest, responseutils.RequestErr, "invalid complex data request type")
		return
	}
	fmt.Printf("\n--- complexDataRequestType %+v, cutoffTime: %+v", complexDataRequestType, cutoffTime)

	fmt.Printf("\n--- finding latest CCLF file based on: CMSID: %+v, cutoffTime: %+v, attributionDate: %+v, fileType: %+v", ad.CMSID, cutoffTime, timeConstraints.AttributionDate, fileType)
	// get latest cclf file
	fmt.Printf("\n--- h.Svc: %+v", h.Svc)
	cclfFileNew, err := h.Svc.GetLatestCCLFFile(
		ctx,
		ad.CMSID,
		// constants.CCLF8FileNum,
		// constants.ImportComplete,
		cutoffTime,
		timeConstraints.AttributionDate,
		fileType,
	)
	if err != nil {
		fmt.Printf("\n--- get latest cclf file error: %+v", err)
		logger.Error(err.Error())
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.DbErr, "")
		return
	}
	fmt.Printf("\n--- cclfFileNew: %+v", cclfFileNew)

	// set PY
	performanceYear := utils.GetPY()
	fmt.Printf("\n--- fileType: %+v, %+v, logic?: %+v", fileType, models.FileTypeRunout, (fileType == models.FileTypeRunout))
	if fileType == models.FileTypeRunout {
		performanceYear -= 1
	}
	fmt.Printf("\n--- performanceYear: %+v", performanceYear)

	// validate cclffile and PY
	if cclfFileNew == nil || performanceYear != cclfFileNew.PerformanceYear {
		fmt.Printf("\n--- failed on validate cclfile/PY, cclfFile: %+v, logic?: %+v", cclfFileNew, (performanceYear != cclfFileNew.PerformanceYear))
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.NotFoundErr, fmt.Sprintf("unable to perform export operations for this Group. No up-to-date attribution information is available for ACOID '%s'. Usually this is due to awaiting new attribution information at the beginning of a Performance Year", ad.CMSID))
		return
	}

	if complexDataRequestType == "GetNewAndExistingBenes" {
		fmt.Println("\n--- start GetNewAndExistingBenes")
		// Retrieve an older CCLF file for beneficiary comparison.
		// This should be older than cclfFileNew AND prior to the "since" parameter, if provided.
		//
		// e.g.
		// - If it’s October 2023
		// - and they request all beneficiary data “since January 1st, 2023"
		// - any beneficiaries added in 2023 are considered "new."
		//
		oldFileUpperBound := rp.Since

		// If the _since parameter is more recent than the latest CCLF file timestamp, set the upper bound
		// for the older file to be prior to the newest file's timestamp.
		if !rp.Since.IsZero() && cclfFileNew.Timestamp.Sub(rp.Since) < 0 {
			oldFileUpperBound = cclfFileNew.Timestamp.Add(-1 * time.Second)
		}

		cclfFileOld, err := h.Svc.GetLatestCCLFFile(
			ctx,
			ad.CMSID,
			// constants.CCLF8FileNum,
			// constants.ImportComplete,
			time.Time{},
			oldFileUpperBound,
			models.FileTypeDefault,
		)
		if err != nil {
			logger.Error(err.Error())
			h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.DbErr, "")
			return
		}
		if cclfFileOld == nil {
			logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW", ad.CMSID, rp.Since)
		} else {
			cclfFileOldID = cclfFileOld.ID
		}
	}
	fmt.Printf("\n--- cclfFileOldID: %+v", cclfFileOldID)

	// ----------------------------------------------------------------------------------------

	fmt.Println("\n--- Start create job")
	newJob.ID, err = h.r.CreateJob(ctx, newJob)
	if err != nil {
		logger.Error("failed to create job: %s", err)
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.DbErr, "")
		return
	}

	if newJob.ID != 0 {
		ctx, logger = log.SetCtxLogger(ctx, "job_id", newJob.ID)
		logger.Info("job id created")
	}

	fmt.Println("\n--- Start pjob")
	// lots of things needed for downstream logic!
	pjob := worker_types.PrepareJobArgs{
		Job:                    newJob,
		ACOID:                  acoID,
		CMSID:                  ad.CMSID,
		CCLFFileNewID:          cclfFileNew.ID,
		CCLFFileOldID:          cclfFileOldID,
		BFDPath:                h.bbBasePath,
		RequestType:            reqType,
		ComplexDataRequestType: complexDataRequestType,
		ResourceTypes:          resourceTypes,
		Since:                  rp.Since,
		CreationTime:           time.Now(),
		ClaimsDate:             timeConstraints.ClaimsDate,
		OptOutDate:             timeConstraints.OptOutDate,
		TransactionID:          r.Context().Value(m.CtxTransactionKey).(string),
	}
	fmt.Printf("\n--- ppreparejob data: %+v", pjob)
	logger.Infof("Adding jobs using %T", h.Enq)
	err = h.Enq.AddPrepareJob(ctx, pjob)
	if err != nil {
		fmt.Printf("\n--- AddPrepareJob err: %+v", err)
		logger.Errorf("failed to add job to the queue: %s", err)
		h.RespWriter.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
		return
	}
	fmt.Printf("\n--- end of bulkrequest, status %+v", w)
	fmt.Printf("\n--- end of bulkrequest, status %+v", r)

	w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/%s/jobs/%d", scheme, r.Host, h.apiVersion, newJob.ID))
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) getResourceTypes(parameters middleware.RequestParameters, cmsID string) []string {
	resourceTypes := parameters.ResourceTypes

	// If caller does not supply resource types, we default to all supported resource types for the specific ACO
	if len(resourceTypes) == 0 {
		if acoConfig, found := h.Svc.GetACOConfigForID(cmsID); found {
			if utils.ContainsString(acoConfig.Data, constants.Adjudicated) {
				resourceTypes = append(resourceTypes, "Patient", "ExplanationOfBenefit", "Coverage")
			}

			if utils.ContainsString(acoConfig.Data, constants.PartiallyAdjudicated) && h.apiVersion != "v1" {
				resourceTypes = append(resourceTypes, "Claim", "ClaimResponse")
			}
		}
	}

	return resourceTypes
}

func (h *Handler) validateResources(resourceTypes []string, cmsID string) error {
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
			(dataType.PartiallyAdjudicated && utils.ContainsString(cfg.Data, constants.PartiallyAdjudicated))
	}

	return false
}

func GetAuthDataFromCtx(r *http.Request) (data auth.AuthData, err error) {
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
