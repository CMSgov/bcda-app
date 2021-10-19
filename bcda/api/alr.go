package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/log"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
)

func (h *Handler) alrRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// retrieve ACO & cms_id data from the context through value
	ad, err := readAuthData(r)
	if err != nil {
		panic("Auth data must be set prior to calling this handler.")
	}

	// retrieve resourceType requested from context
	rp, ok := middleware.RequestParametersFromContext(ctx)
	if !ok {
		panic("Request parameters must be set prior to calling this handler.")
	}

	// Currently, we don't do anything with resource types, thus commented out
	//if err := h.validateRequest(rp.ResourceTypes, ad.CMSID); err != nil {
		//oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr,
			//err.Error())
		//responseutils.WriteError(oo, w, http.StatusBadRequest)
		//return
	//}

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

	// Use a transaction to guarantee that the job only gets created if we queue all of the alrJobs
	rtx := postgres.NewRepositoryTx(tx)

	alrMBIs, err := h.r.GetAlrMBIs(ctx, ad.CMSID)
	if err != nil {
		return // Rollback handled in defer
	}

	// Take slices of MBIs into Jobs
	alrJobs := h.Svc.GetAlrJobs(ctx, alrMBIs)

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
