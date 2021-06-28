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
	"github.com/CMSgov/bcda-app/log"
	"github.com/pborman/uuid"
)

var (
	repository   models.Repository
	jobTimeout   time.Duration
	retrySeconds int
)

func init() {
	repository = postgres.NewRepository(database.Connection)
	jobTimeout = time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))
	retrySeconds = utils.GetEnvInt("CLIENT_RETRY_AFTER_IN_SECONDS", 0)
}

func CheckConcurrentJobs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
		if !ok {
			panic("AuthData should be set before calling this handler")
		}

		rp, ok := RequestParametersFromContext(r.Context())
		if !ok {
			panic("RequestParameters should be set before calling this handler")
		}

		version, err := getVersion(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rw, err := getRespWriter(version)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		acoID := uuid.Parse(ad.ACOID)

		pendingAndInProgressJobs, err := repository.GetJobs(r.Context(), acoID, models.JobStatusInProgress, models.JobStatusPending)
		if err != nil {
			log.API.Error(fmt.Errorf("failed to lookup pending and in-progress jobs: %w", err))
			rw.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}
		if len(pendingAndInProgressJobs) > 0 {
			if hasDuplicates(pendingAndInProgressJobs, rp.ResourceTypes, rp.Version) {
				w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func hasDuplicates(pendingAndInProgressJobs []*models.Job, types []string, version string) bool {
	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	allResources := len(types) == 0

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

		// Any in-progress job will have duplicate types since the caller
		// is requesting all resources
		if allResources {
			return true
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

	// No duplicates
	return false
}
