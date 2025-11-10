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
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"net/http"
	"time"

	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	responseutilsv2 "github.com/CMSgov/bcda-app/bcda/responseutils/v2"
	responseutilsv3 "github.com/CMSgov/bcda-app/bcda/responseutils/v3"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	m "github.com/CMSgov/bcda-app/middleware"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	JobTimeout time.Duration
	Enq        queueing.Enqueuer
	Svc        service.Service

	supportedDataTypes     map[string]service.ClaimType
	supportedResourceTypes []string
	supportedStatuses      map[models.JobStatus]struct{}
	bbBasePath             string
	apiVersion             string
	RespWriter             fhirResponseWriter
	// cfg                    *service.Config

	// Needed to have access to the repository/db for lookup needed in the bulkRequest.
	// TODO (BCDA-3412): Remove this reference once we've captured all of the necessary
	// logic into a service method.
	r models.Repository
}

type fhirResponseWriter interface {
	Exception(context.Context, http.ResponseWriter, int, string, string)
	NotFound(context.Context, http.ResponseWriter, int, string, string)
	JobsBundle(context.Context, http.ResponseWriter, []*models.Job, string)
}

func NewHandler(dataTypes map[string]service.ClaimType, basePath string, apiVersion string, db *sql.DB, pool *pgxv5Pool.Pool) *Handler {
	return newHandler(dataTypes, basePath, apiVersion, db, pool)
}

func newHandler(dataTypes map[string]service.ClaimType, basePath string, apiVersion string, db *sql.DB, pool *pgxv5Pool.Pool) *Handler {
	h := &Handler{JobTimeout: time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))}

	h.Enq = queueing.NewEnqueuer(db, pool)

	cfg, err := service.LoadConfig()
	if err != nil {
		log.API.Fatalf("Failed to load service config. Err: %v", err)
	}
	if len(cfg.ACOConfigs) == 0 {
		log.API.Fatalf("no ACO configs found, these are required for processing logic")
	}

	repository := postgres.NewRepository(db)
	h.r = repository
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
		h.RespWriter = responseutils.NewFhirResponseWriter()
	case "v2":
		h.RespWriter = responseutilsv2.NewFhirResponseWriter()
	case constants.V3Version:
		h.RespWriter = responseutilsv3.NewFhirResponseWriter()
	default:
		log.API.Fatalf("unexpected API version: %s", h.apiVersion)
	}

	return h
}

func (h *Handler) BulkPatientRequest(w http.ResponseWriter, r *http.Request) {
	reqType := constants.DefaultRequest // historical data for new beneficiaries will not be retrieved (this capability is only available with /Group)
	h.bulkRequest(w, r, reqType)
}

func (h *Handler) BulkGroupRequest(w http.ResponseWriter, r *http.Request) {
	const (
		groupAll    = "all"
		groupRunout = "runout"
	)
	reqType := constants.DefaultRequest
	groupID := chi.URLParam(r, "groupId")
	ctx := r.Context()

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
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Invalid group ID (%+v)", responseutils.RequestErr, groupID),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, "Invalid group ID")
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
	statusTypes = models.AllJobStatuses // default request to retrieve jobs with all statuses
	params, ok := r.URL.Query()["_status"]
	ctx := r.Context()

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
				ctx, _ = log.ErrorExtra(
					ctx,
					fmt.Sprintf("%s: %+v", responseutils.RequestErr, errMsg),
					logrus.Fields{"resp_status": http.StatusBadRequest},
				)
				h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, errMsg)
				return
			}
		}

		// validate status types provided match our valid list of statuses
		if err = h.validateStatuses(statusTypes); err != nil {
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: %+v", responseutils.RequestErr, err),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
			return
		}
	}

	if ad, err = GetAuthDataFromCtx(r); err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: %+v", responseutils.TokenErr, err),
			logrus.Fields{"resp_status": http.StatusUnauthorized},
		)
		h.RespWriter.Exception(ctx, w, http.StatusUnauthorized, responseutils.TokenErr, "")
		return
	}

	jobs, err := h.Svc.GetJobs(ctx, uuid.Parse(ad.ACOID), statusTypes...)
	if err != nil {
		if ok := goerrors.As(err, &service.JobsNotFoundError{}); ok {
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: %+v", responseutils.DbErr, err),
				logrus.Fields{"resp_status": http.StatusNotFound},
			)
			h.RespWriter.Exception(ctx, w, http.StatusNotFound, responseutils.DbErr, err.Error())
		} else {
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: %+v", responseutils.InternalErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError},
			)
			h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "")
		}
	}

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	host := fmt.Sprintf("%s://%s", scheme, r.Host)

	// pass in the ctx here and log with the ctx logger
	h.RespWriter.JobsBundle(ctx, w, jobs, host)
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
	ctx := r.Context()

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: %+v", responseutils.RequestErr, err),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		//We don't need to return the full error to a consumer.
		//We pass a bad request header (400) for this exception due to the inputs always being invalid for our purposes
		h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, "could not parse job id")

		return
	}

	job, jobKeys, err := h.Svc.GetJobAndKeys(ctx, uint(jobID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Requested job not found.  Error: %+v", responseutils.DbErr, err),
				logrus.Fields{"resp_status": http.StatusNotFound, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusNotFound, responseutils.DbErr, "Job not found.")
		} else {
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: Error attempting to request job. Error: %+v", responseutils.InternalErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "Error trying to fetch job. Please try again.")
		}

		return
	}

	switch job.Status {

	case models.JobStatusFailed, models.JobStatusFailedExpired:
		logger.Error(job.Status)
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Status: %+v", responseutils.JobFailed, job.Status),
			logrus.Fields{"resp_status": http.StatusInternalServerError, "job_id": jobID},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.JobFailed, responseutils.DetailJobFailed)
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
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Job is expired but was not archived in time", responseutils.NotFoundErr),
				logrus.Fields{"resp_status": http.StatusGone, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusGone, responseutils.NotFoundErr, "")
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
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: Error marshaling response body: %+v", responseutils.InternalErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: Error writing response body: %+v", responseutils.InternalErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}

		w.WriteHeader(http.StatusOK)
	case models.JobStatusArchived, models.JobStatusExpired:
		w.Header().Set("Expires", job.UpdatedAt.Add(h.JobTimeout).String())
		ctx, _ = log.WarnExtra(
			ctx,
			fmt.Sprintf("%s: Job is Archived or Expired", responseutils.NotFoundErr),
			logrus.Fields{"resp_status": http.StatusGone, "job_id": jobID},
		)
		h.RespWriter.Exception(ctx, w, http.StatusGone, responseutils.NotFoundErr, "")
	case models.JobStatusCancelled, models.JobStatusCancelledExpired:
		h.RespWriter.NotFound(ctx, w, http.StatusNotFound, responseutils.NotFoundErr, "Job has been cancelled.")

	}
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")
	ctx := r.Context()

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "cannot convert jobID to uint")
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: %+v", responseutils.RequestErr, err),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
		return
	}

	_, err = h.Svc.CancelJob(ctx, uint(jobID))
	if err != nil {
		switch err {
		case service.ErrJobNotCancellable:
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Job is not cancellable", responseutils.DeletedErr),
				logrus.Fields{"resp_status": http.StatusGone, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusGone, responseutils.DeletedErr, err.Error())
			return
		default:
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: %+v", responseutils.DbErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError, "job_id": jobID},
			)
			h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.DbErr, err.Error())
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
	var (
		ad   auth.AuthData
		err  error
		resp AttributionFileStatusResponse
	)

	if ad, err = GetAuthDataFromCtx(r); err != nil {
		ctx, _ = log.WarnExtra(
			ctx,
			fmt.Sprintf("%s: %+v", responseutils.TokenErr, err),
			logrus.Fields{"resp_status": http.StatusUnauthorized},
		)
		h.RespWriter.Exception(ctx, w, http.StatusUnauthorized, responseutils.TokenErr, "")
		return
	}

	// Retrieve the most recent cclf 8 file we have successfully ingested
	group := chi.URLParam(r, "groupId")
	notFoundMsg := fmt.Sprintf("Unable to perform export operations for this Group. No up-to-date attribution information is available for Group '%s'. Usually this is due to awaiting new attribution information at the beginning of a Performance Year.", group)
	asd, err := h.getAttributionFileStatus(ctx, ad.CMSID, models.FileTypeDefault)
	if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok {
		ctx, _ = log.WarnExtra(
			ctx,
			fmt.Sprintf("%s: %+v Error: %+v", responseutils.NotFoundErr, notFoundMsg, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.NotFoundErr, notFoundMsg)
		return
	}
	if asd != nil {
		resp.Data = append(resp.Data, *asd)
	}

	// Retrieve the most recent cclf 8 runout file we have successfully ingested
	asr, err := h.getAttributionFileStatus(ctx, ad.CMSID, models.FileTypeRunout)
	if ok := goerrors.As(err, &service.CCLFNotFoundError{}); ok {
		ctx, _ = log.WarnExtra(
			ctx,
			fmt.Sprintf("%s: %+v Error: %+v", responseutils.NotFoundErr, notFoundMsg, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.NotFoundErr, notFoundMsg)
		return
	}
	if asr != nil {
		resp.Data = append(resp.Data, *asr)
	}

	if resp.Data == nil {
		ctx, _ = log.WarnExtra(
			ctx,
			fmt.Sprintf("%s: Could not find any CCLF8 file", responseutils.NotFoundErr),
			logrus.Fields{"resp_status": http.StatusNotFound},
		)
		h.RespWriter.Exception(ctx, w, http.StatusNotFound, responseutils.NotFoundErr, "")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Failed to encode JSON response: %+v", responseutils.RequestErr, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.RequestErr, "")
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
	// Create context to encapsulate the entire workflow. In the future, we can define child context's for timing.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	logger := log.GetCtxLogger(ctx)

	var (
		ad  auth.AuthData
		err error
	)

	if ad, err = GetAuthDataFromCtx(r); err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: %+v", responseutils.TokenErr, err),
			logrus.Fields{"resp_status": http.StatusUnauthorized},
		)
		h.RespWriter.Exception(ctx, w, http.StatusUnauthorized, responseutils.TokenErr, "")
		return
	}

	rp, ok := middleware.GetRequestParamsFromCtx(ctx)
	if !ok {
		panic("Request parameters must be set prior to calling this handler.")
	}

	resourceTypes := h.getResourceTypes(rp, ad.CMSID)
	if err = h.validateResources(resourceTypes, ad.CMSID); err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Error validating resources: %+v", responseutils.RequestErr, err),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, err.Error())
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

	fileType := models.FileTypeDefault
	if reqType == constants.Runout {
		fileType = models.FileTypeRunout
	}

	timeConstraints, err := h.Svc.GetTimeConstraints(ctx, ad.CMSID)
	if err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Error setting time constraints: %+v", responseutils.InternalErr, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "")
		return
	}

	cutoffTime, complexDataRequestType := h.Svc.GetCutoffTime(ctx, reqType, rp.Since, timeConstraints, fileType)
	if complexDataRequestType == "" {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Invalid complex data request type: %+v", responseutils.RequestErr, err),
			logrus.Fields{"resp_status": http.StatusBadRequest},
		)
		h.RespWriter.Exception(ctx, w, http.StatusBadRequest, responseutils.RequestErr, "invalid complex data request type")
		return
	}

	cclfFileNew, err := h.Svc.GetLatestCCLFFile(
		ctx,
		ad.CMSID,
		cutoffTime,
		timeConstraints.AttributionDate,
		fileType,
	)
	if err != nil {
		if errors.As(err, &service.CCLFNotFoundError{}) {
			// Check if this is an expired runout data request
			if fileType == models.FileTypeRunout && !cutoffTime.IsZero() {
				msg := "runout data is no longer available. Runout data expires 180 days after ingestion"
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: %+v", responseutils.NotFoundErr, msg),
					logrus.Fields{"resp_status": http.StatusNotFound},
				)
				h.RespWriter.Exception(ctx, w, http.StatusNotFound, responseutils.NotFoundErr, msg)
				return
			}

			msg := "failed to start job; attribution file not found"
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: %+v", responseutils.NotFoundErr, msg),
				logrus.Fields{"resp_status": http.StatusNotFound},
			)
			h.RespWriter.Exception(ctx, w, http.StatusNotFound, responseutils.NotFoundErr, msg)
			return
		}

		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Failed to retrieve latest cclf file: %+v", responseutils.DbErr, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.DbErr, responseutils.DbErr)
		return
	}

	performanceYear := utils.GetPY()
	if fileType == models.FileTypeRunout {
		performanceYear -= 1
	}

	// validate cclffile and PY
	if cclfFileNew == nil || performanceYear != cclfFileNew.PerformanceYear {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Failed to validate cclf file or performance year of found cclf file", responseutils.NotFoundErr),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.NotFoundErr, fmt.Sprintf("unable to perform export operations for this Group. No up-to-date attribution information is available for ACOID '%s'. Usually this is due to awaiting new attribution information at the beginning of a Performance Year", ad.CMSID))
		return
	}

	var cclfFileOldID uint
	if complexDataRequestType == constants.GetNewAndExistingBenes {
		cclfFileOldID, err = h.Svc.FindOldCCLFFile(ctx, ad.CMSID, rp.Since, cclfFileNew.Timestamp)
		if err != nil {
			if errors.As(err, &service.CCLFNotFoundError{}) {
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: CCLF file not found for given _since parameter: %s", responseutils.NotFoundErr, rp.Since.String()),
					logrus.Fields{"resp_status": http.StatusNotFound},
				)
				h.RespWriter.Exception(ctx, w, http.StatusNotFound, responseutils.NotFoundErr, "failed to start job; attribution file not found for given _since parameter.")
				return
			}

			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: failed to retrieve cclf file: %+v", responseutils.DbErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError},
			)
			h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.DbErr, "")
			return
		}
	}

	newJob.ID, err = h.r.CreateJob(ctx, newJob)
	if err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Failed to create job: %+v", responseutils.DbErr, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.DbErr, "")
		return
	}

	if newJob.ID != 0 {
		ctx, logger = log.SetCtxLogger(ctx, "job_id", newJob.ID)
		logger.Info("job id created")
	}

	// lots of things needed for downstream logic!
	prepJob := worker_types.PrepareJobArgs{
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
		TypeFilter:             rp.TypeFilter,
		CreationTime:           time.Now(),
		ClaimsDate:             timeConstraints.ClaimsDate,
		OptOutDate:             timeConstraints.OptOutDate,
		TransactionID:          ctx.Value(m.CtxTransactionKey).(string),
	}

	logger.Infof("Adding jobs using %T", h.Enq)
	err = h.Enq.AddPrepareJob(ctx, prepJob)
	if err != nil {
		ctx, _ = log.ErrorExtra(
			ctx,
			fmt.Sprintf("%s: Failed to add job to the queue: %+v", responseutils.InternalErr, err),
			logrus.Fields{"resp_status": http.StatusInternalServerError},
		)
		h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "")
		return
	}

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

			if utils.ContainsString(acoConfig.Data, constants.PartiallyAdjudicated) && h.apiVersion == "v2" {
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

func (h *Handler) authorizedResourceAccess(dataType service.ClaimType, cmsID string) bool {
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
