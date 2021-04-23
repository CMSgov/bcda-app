package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/log"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
)

func (h *Handler) alrRequest(w http.ResponseWriter, r *http.Request, reqType service.RequestType) {
	ctx := r.Context()

	ad, err := readAuthData(r)
	if err != nil {
		panic("Auth data must be set prior to calling this handler.")
	}

	rp, ok := middleware.RequestParametersFromContext(ctx)
	if !ok {
		panic("Request parameters must be set prior to calling this handler.")
	}

	if err := h.validateRequest(rp.ResourceTypes); err != nil {
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr,
			err.Error())
		responseutils.WriteError(oo, w, http.StatusBadRequest)
		return
	}

	req := service.DefaultAlrRequest
	if reqType == service.Runout {
		req = service.RunoutAlrRequest
	}

	// Depending on how the request is sent to the handler,
	// the r.URL.Scheme may be unset.
	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}

	requestURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
	if r.URL.RawQuery != "" {
		requestURL = fmt.Sprintf("%s?%s", requestURL, r.URL.RawQuery)
	}

	newJob := models.Job{
		ACOID:           uuid.Parse(ad.ACOID),
		RequestURL:      requestURL,
		Status:          models.JobStatusPending,
		TransactionTime: time.Now(),
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("failed to start transaction: %w", err)
		log.API.Error(err.Error())
		oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
			responseutils.InternalErr, err.Error())
		responseutils.WriteError(oo, w, http.StatusInternalServerError)
		return
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.API.Warnf("Failed to rollback transaction %s", err.Error())
			}
			log.API.Errorf("Could not handle ALR request %s", err.Error())
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.DbErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			log.API.Errorf("Failed to commit transaction %s", err.Error())
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.DbErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/v1/jobs/%d", scheme, r.Host, newJob.ID))
		w.WriteHeader(http.StatusAccepted)
	}()

	// Use a transaction to guarantee that the job only gets created iff we queue all of the alrJobs
	rtx := postgres.NewRepositoryTx(tx)

	alrJobs, err := h.Svc.GetAlrJobs(ctx, ad.CMSID, req, service.AlrRequestWindow{LowerBound: rp.Since})
	if err != nil {
		return // Rollback handled in defer
	}

	newJob.JobCount = len(alrJobs)
	newJob.ID, err = rtx.CreateJob(ctx, newJob)
	if err != nil {
		return // Rollback handled in the defer
	}

	for _, j := range alrJobs {
		j.ID = newJob.ID
		priority := h.Svc.GetJobPriority(ad.CMSID, "alr", !rp.Since.IsZero())
		if err = h.Enq.AddAlrJob(*j, int(priority)); err != nil {
			return // Rollback handled in the defer
		}
	}

	// Commit handled in defer
}

// Since we are overloading the Patient resource, we require the caller to provide a typeFilter
// to specify an ALR resource.
func isALRRequest(r *http.Request) bool {
	typeParam := r.URL.Query().Get("_type")
	typeFilterParam := r.URL.Query().Get("_typeFilter")

	hasType := typeParam == "Patient,Observation" ||
		typeParam == "Observation,Patient"
	hasTypeFilter := typeFilterParam == "Patient?profile=ALR,Observation?profile=ALR" ||
		typeFilterParam == "Observation?profile=ALR,Patient?profile=ALR"

	return hasType && hasTypeFilter
}
