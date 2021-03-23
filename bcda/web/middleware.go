package web

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ConnectionClose(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		next.ServeHTTP(w, r)
	})
}

func SecurityHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if servicemux.IsHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			w.Header().Set("Cache-Control", "no-cache; no-store; must-revalidate; max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}

func ConcurrentJobs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acoID := uuid.NewRandom()
		pendingAndInProgressJobs, err := repository.GetJobs(r.Context(), acoID, models.JobStatusInProgress, models.JobStatusPending)
		if err != nil {
			log.Error(fmt.Errorf("failed to lookup pending and in-progress jobs: %w", err))
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, "")
			responseutils.WriteError(oo, w, http.StatusInternalServerError)
		}
		if len(pendingAndInProgressJobs) > 0 {
			if types, err := check429(pendingAndInProgressJobs, resourceTypes, version); err != nil {
				if _, ok := err.(duplicateTypeError); ok {
					w.Header().Set("Retry-After", strconv.Itoa(utils.GetEnvInt("CLIENT_RETRY_AFTER_IN_SECONDS", 0)))
					w.WriteHeader(http.StatusTooManyRequests)
				} else {
					log.Error(err)
					oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
						responseutils.InternalErr, "")
					responseutils.WriteError(oo, w, http.StatusInternalServerError)
				}

				return
			}
		}
	})
}

func checkDuplicates(pendingAndInProgressJobs []*models.Job, types []string, version string) ([]string, error) {
	var unworkedTypes []string

	for _, t := range types {
		worked := false
		for _, job := range pendingAndInProgressJobs {
			req, err := url.Parse(job.RequestURL)
			if err != nil {
				return nil, err
			}
			jobVersion, err := getVersion(req)
			if err != nil {
				return nil, err
			}

			// We allow different API versions to trigger jobs with the same resource type
			if jobVersion != version {
				continue
			}

			// If the job has timed-out we will allow new job to be created
			if time.Now().After(job.CreatedAt.Add(GetJobTimeout())) {
				continue
			}

			if requestedTypes, ok := req.Query()["_type"]; ok {
				// if this type is being worked no need to keep looking, break out and go to the next type.
				if strings.Contains(requestedTypes[0], t) {
					worked = true
					break
				}
			} else {
				// we have an export all types that is still in progress
				return nil, duplicateTypeError{}
			}
		}
		if !worked {
			unworkedTypes = append(unworkedTypes, t)
		}
	}
	if len(unworkedTypes) == 0 {
		return nil, duplicateTypeError{}
	} else {
		return unworkedTypes, nil
	}
}

func getVersion(url *url.URL) (string, error) {
	re := regexp.MustCompile(`\/api\/(.*)\/[Patient|Group].*`)
	parts := re.FindStringSubmatch(url.Path)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected path provided %s", url.Path)
	}
	return parts[1], nil
}
