package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	mw "github.com/CMSgov/bcda-app/middleware"
	"github.com/pborman/uuid"
)

func SharedExportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// logger := log.GetCtxLogger(ctx)
	var err error

	db := database.Connect()
	pool := database.ConnectPool()
	repo := postgres.NewRepository(db)

	enq := queueing.NewEnqueuer(db, pool)

	// validate request headers

	since, ok := r.URL.Query()["_since"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing _since parameter"))
		return
	}
	sinceDate, err := time.Parse(time.RFC3339Nano, since[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid _since parameter"))
		return
	}

	version, ok := r.URL.Query()["_version"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing _version parameter"))
		return
	}
	resourceTypes, ok := r.URL.Query()["_resourceTypes"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing _resourceTypes parameter"))
		return
	}
	partnerID, ok := r.URL.Query()["_partnerID"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing _partnerID parameter"))
		return
	}
	dataTypes, ok := r.URL.Query()["_dataTypes"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing _dataTypes parameter"))
		return
	}
	mbis, ok := r.URL.Query()["_mbis"] // TODO: change this to a POST request and pass this as body
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing _mbis parameter"))
		return
	}

	var bfdPath string
	switch version[0] {
	case "v1":
		bfdPath = "/v1/fhir"
	case "v2":
		bfdPath = "/v2/fhir"
	case "v3":
		bfdPath = "/v3/fhir"
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid _version parameter"))
		return
	}

	newJob := models.Job{
		ACOID:      uuid.Parse("0c527d2e-2e8a-4808-b11d-0fa06baf8254"), // A9994, TODO: this needs to not be null for the jobs record in the DB
		RequestURL: fmt.Sprintf("https://%s%s", r.Host, r.URL),
		Status:     models.JobStatusPending,
	}

	newJob.ID, err = repo.CreateJob(ctx, newJob)
	if err != nil {
		// ctx, _ = log.WriteErrorWithFields(
		// 	ctx,
		// 	fmt.Sprintf("%s: Failed to create job: %+v", responseutils.DbErr, err),
		// 	logrus.Fields{"resp_status": http.StatusInternalServerError},
		// )
		// h.RespWriter.Exception(ctx, w, http.StatusInternalServerError, responseutils.DbErr, "")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to create job"))
		return
	}

	// if newJob.ID != 0 {
	// 	ctx, _ = log.WriteInfoWithFields(
	// 		ctx,
	// 		fmt.Sprintf("job id created: %d", newJob.ID),
	// 		logrus.Fields{"job_id": newJob.ID},
	// 	)
	// }

	// lots of things needed for downstream logic!
	prepJob := worker_types.PrepareSharedJobArgs{
		Job: newJob,
		// ACOID:                  acoID,
		// CMSID:                  ad.CMSID,
		PartnerID:     partnerID[0],
		BFDPath:       bfdPath,
		ResourceTypes: resourceTypes,
		Since:         sinceDate,
		CreationTime:  time.Now(),
		TransactionID: ctx.Value(mw.CtxTransactionKey).(string),
		MBIs:          mbis, // TODO: we save internal Ids for the cclf_beneficiaries into our river_job record for our normal job process, is it problematic to save mbis? if so we will need to save into the DB?  or pull from file later on?
		DataTypes:     dataTypes,
	}

	err = enq.AddPrepareSharedJob(ctx, prepJob)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed to add job to the queue: %v", err)))
		return
	}

	w.Header().Set("Content-Location", fmt.Sprintf("https://%s/api/%s/jobs/%d", r.Host, version[0], newJob.ID))
	w.WriteHeader(http.StatusAccepted)

}
