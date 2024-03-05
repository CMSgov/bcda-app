package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/log"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
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

		rw, _ := getResponseWriterFromRequestPath(w, r)
		if rw == nil {
			return
		}

		acoID := uuid.Parse(ad.ACOID)
		pendingAndInProgressJobs, err := repository.GetJobs(r.Context(), acoID, models.JobStatusInProgress, models.JobStatusPending)
		if err != nil {
			logger := log.GetCtxLogger(r.Context())
			logger.Error(fmt.Errorf("failed to lookup pending and in-progress jobs: %w", err))
			rw.Exception(r.Context(), w, http.StatusInternalServerError, responseutils.InternalErr, "")
			return
		}
		if len(pendingAndInProgressJobs) > 0 {
			if hasDuplicates(r.Context(), pendingAndInProgressJobs, rp.ResourceTypes, rp.Version, rp.RequestURL) {
				w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func hasDuplicates(ctx context.Context, pendingAndInProgressJobs []*models.Job, types []string, version string, newRequestUrl string) bool {
	logger := log.GetCtxLogger(ctx)
	if strings.Contains(newRequestUrl, "/jobs") && !strings.Contains(newRequestUrl, "$export") {
		return false
	}

	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	allResources := len(types) == 0

	for _, job := range pendingAndInProgressJobs {
		logger.Infof("Checking if new request is duplicate of pending or in-progress job %d\n", job.ID)

		// Cannot determine duplicates if we can't get the underlying URL
		req, err := url.Parse(job.RequestURL)
		if err != nil {
			logger.Warn(errors.Wrap(err, "Could not parse job request URL to determine duplicates -- ignoring existing job"))
			continue
		}

		// Cannot determine duplicates if we can't figure out the version
		jobVersion, err := getVersion(req.Path)
		if err != nil {
			logger.Warn(errors.Wrap(err, "Could not parse job's API version to determine duplicates -- ignoring existing job"))
			continue
		}

		// We allow different API versions to trigger jobs with the same resource type
		if jobVersion != version {
			logger.Info("Existing Job version differs from new job version -- ignoring existing job")
			continue
		}

		// If the job has timed-out we will allow new job to be created
		if time.Now().After(job.CreatedAt.Add(jobTimeout)) {
			logger.Info("Existing job timed out -- ignoring existing job")
			continue
		}

		// Ensure that the requestUrls are not the same
		if job.RequestURL == newRequestUrl {
			logger.Info("New request has the same requestUrl as existing job -- disallowing request")
			return true
		}

		// Any in-progress job will have duplicate types since the caller
		// is requesting all resources
		if allResources {
			logger.Info("New request is for all resources and will overlap with existing job -- disallowing request")
			return true
		}

		if requestedTypes, ok := req.Query()["_type"]; ok {
			for _, rt := range requestedTypes {
				if _, ok := typeSet[rt]; ok {
					logger.Info("New request types overlap with existing job types -- disallowing request")
					return true
				}
			}
		} else {
			// we have an export all types that is still in progress
			logger.Info("Existing job is exporting all types -- disallowing request")
			return true
		}
	}

	// No duplicates
	logger.Info("No duplicate jobs exist for incoming request -- allowing request")
	return false
}
