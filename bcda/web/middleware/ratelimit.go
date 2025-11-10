package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/log"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RateLimitMiddleware struct {
	config       *service.Config
	repository   models.Repository
	jobTimeout   time.Duration
	retrySeconds int
}

func NewRateLimitMiddleware(config *service.Config, db *sql.DB) RateLimitMiddleware {
	r := postgres.NewRepository(db)
	jt := time.Hour * time.Duration(utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24))
	rs := utils.GetEnvInt("CLIENT_RETRY_AFTER_IN_SECONDS", 0)
	return RateLimitMiddleware{config: config, repository: r, jobTimeout: jt, retrySeconds: rs}
}

func (m RateLimitMiddleware) CheckConcurrentJobs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ad, ok := ctx.Value(auth.AuthDataContextKey).(auth.AuthData)
		if !ok {
			panic("AuthData should be set before calling this handler")
		}

		rp, ok := GetRequestParamsFromCtx(ctx)
		if !ok {
			panic("RequestParameters should be set before calling this handler")
		}

		rw, _ := getResponseWriterFromRequestPath(w, r)
		if rw == nil {
			return
		}

		acoID := uuid.Parse(ad.ACOID)

		if shouldRateLimit(m.config.RateLimitConfig, ad.CMSID) {
			pendingAndInProgressJobs, err := m.repository.GetJobs(ctx, acoID, models.JobStatusInProgress, models.JobStatusPending)
			if err != nil {
				ctx, _ = log.ErrorExtra(
					ctx,
					fmt.Sprintf("%s: Failed to lookup pending and in-progress jobs %+v", responseutils.InternalErr, err),
					logrus.Fields{"resp_status": http.StatusInternalServerError},
				)
				rw.Exception(ctx, w, http.StatusInternalServerError, responseutils.InternalErr, "")
				return
			}
			if len(pendingAndInProgressJobs) > 0 {
				if m.hasDuplicates(ctx, pendingAndInProgressJobs, rp.ResourceTypes, rp.Version, rp.RequestURL) {
					w.Header().Set("Retry-After", strconv.Itoa(m.retrySeconds))
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func shouldRateLimit(config service.RateLimitConfig, cmsID string) bool {
	if config.All || slices.Contains(config.ACOs, cmsID) {
		return true
	}
	return false
}

func (m RateLimitMiddleware) hasDuplicates(ctx context.Context, pendingAndInProgressJobs []*models.Job, types []string, version string, newRequestUrl string) bool {
	logger := log.GetCtxLogger(ctx)

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
		if time.Now().After(job.CreatedAt.Add(m.jobTimeout)) {
			logger.Info("Existing job timed out -- ignoring existing job")
			continue
		}
		//Error is ignored here due to url.Parse handling above.
		unescapedOldJobURL, _ := url.QueryUnescape(req.Path + "?" + req.RawQuery)

		unescapedNewJobURL, err := url.QueryUnescape(newRequestUrl)
		if err != nil {
			logger.Info("Unable to unescape new job URL -- ignoring new job")
			continue
		}
		// Ensure that the requestUrls are not the same
		if unescapedOldJobURL == unescapedNewJobURL {
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
