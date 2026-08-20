package api

import (
	"encoding/json"
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

type SharedData struct {
	Since         string   `json:"since"`
	Version       string   `json:"version"`
	ResourceTypes []string `json:"resource_types"`
	PartnerID     string   `json:"partner_id"`
	DataTypes     []string `json:"data_types"`
	MBIs          []string `json:"mbis"`
}

func SharedExportPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// logger := log.GetCtxLogger(ctx)
	var err error

	// TODO validate request headers?

	db := database.Connect()
	pool := database.ConnectPool()
	repo := postgres.NewRepository(db)

	enq := queueing.NewEnqueuer(db, pool)

	var payload SharedData

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Since == "" || payload.Version == "" || len(payload.ResourceTypes) == 0 || payload.PartnerID == "" || len(payload.DataTypes) == 0 || len(payload.MBIs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "missing required fields in payload", http.StatusBadRequest)
		return
	}

	sinceDate, err := time.Parse(time.RFC3339Nano, payload.Since)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "invalid _since parameter", http.StatusBadRequest)
		return
	}

	var bfdPath string
	switch payload.Version {
	case "v1":
		bfdPath = "/v1/fhir"
	case "v2":
		bfdPath = "/v2/fhir"
	case "v3":
		bfdPath = "/v3/fhir"
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("invalid _version parameter"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	newJob := models.Job{
		ACOID:      uuid.Parse("0c527d2e-2e8a-4808-b11d-0fa06baf8254"), // A9994, TODO: this needs to not be null for the jobs record in the DB                                         // TODO: this needs to not be null for the jobs record in the DB
		RequestURL: fmt.Sprintf("https://%s%s", r.Host, r.URL),
		Status:     models.JobStatusPending,
	}

	newJob.ID, err = repo.CreateJob(ctx, newJob)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("failed to create job"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	prepJob := worker_types.PrepareSharedJobArgs{
		Job: newJob,
		// ACOID:                  acoID,
		// CMSID:                  ad.CMSID,
		PartnerID:     payload.PartnerID,
		BFDPath:       bfdPath,
		ResourceTypes: payload.ResourceTypes,
		Since:         sinceDate,
		CreationTime:  time.Now(),
		TransactionID: ctx.Value(mw.CtxTransactionKey).(string),
		MBIs:          payload.MBIs, // TODO: we save internal Ids for the cclf_beneficiaries into our river_job record for our normal job process, is it problematic to save mbis? if so we will need to save into the DB?  or pull from file later on?
		DataTypes:     payload.DataTypes,
	}

	err = enq.AddPrepareSharedJob(ctx, prepJob)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to add job to the queue: %v", err)
		return
	}

	w.Header().Set("Content-Location", fmt.Sprintf("https://%s/api/%s/jobs/%d", r.Host, payload.Version, newJob.ID))
	w.WriteHeader(http.StatusAccepted)
}
