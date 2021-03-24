package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

var (
	repository models.Repository
	jobTimeout time.Duration
)

func init() {
	repository = postgres.NewRepository(database.Connection)
	jobTimeout = time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))
}

func CheckConcurrentJobs(next Handler) Handler {
	return Handler(func(w http.ResponseWriter, r *http.Request, rp RequestParameters) {
		ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
		if !ok {
			panic("AuthData should be set before calling this handler")
		}
		acoID := uuid.Parse(ad.ACOID)

		pendingAndInProgressJobs, err := repository.GetJobs(r.Context(), acoID, models.JobStatusInProgress, models.JobStatusPending)
		if err != nil {
			log.Error(fmt.Errorf("failed to lookup pending and in-progress jobs: %w", err))
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
		}
		if len(pendingAndInProgressJobs) > 0 {
			if hasDuplicates(pendingAndInProgressJobs, rp.ResourceTypes, rp.Version) {
				w.Header().Set("Retry-After", strconv.Itoa(utils.GetEnvInt("CLIENT_RETRY_AFTER_IN_SECONDS", 0)))
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}
		next(w, r, rp)
	})
}

func hasDuplicates(pendingAndInProgressJobs []*models.Job, types []string, version string) bool {
	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	for _, job := range pendingAndInProgressJobs {
		// Cannot determine duplicates if we can't get the underlying URL
		req, err := url.Parse(job.RequestURL)
		if err != nil {
			continue
		}

		// Cannot determine duplicates if we can't figure out the version
		jobVersion, err := getVersion(req.Path)
		if err != nil {
			continue
		}

		// We allow different API versions to trigger jobs with the same resource type
		if jobVersion != version {
			continue
		}

		// If the job has timed-out we will allow new job to be created
		if time.Now().After(job.CreatedAt.Add(jobTimeout)) {
			continue
		}

		if requestedTypes, ok := req.Query()["_type"]; ok {
			for _, rt := range requestedTypes {
				if _, ok := typeSet[rt]; ok {
					return true
				}
			}
		} else {
			// we have an export all types that is still in progress
			return true
		}
	}
}
