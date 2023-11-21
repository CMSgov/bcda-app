package logging

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/log"
)

// https://github.com/go-chi/chi/blob/master/_examples/logging/main.go

func NewStructuredLogger() func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{Logger: log.Request})
}

type StructuredLogger struct {
	Logger logrus.FieldLogger
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &log.StructuredLoggerEntry{Logger: l.Logger}
	logFields := logrus.Fields{}

	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields["request_id"] = reqID
	}

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	logFields["http_scheme"] = scheme
	logFields["http_proto"] = r.Proto
	logFields["http_method"] = r.Method

	logFields["remote_addr"] = r.RemoteAddr
	logFields["forwarded_for"] = r.Header.Get("X-Forwarded-For")
	logFields["user_agent"] = r.UserAgent()
	logFields["accept_encoding"] = r.Header.Get("Accept-Encoding")

	logFields["uri"] = fmt.Sprintf("%s://%s%s", scheme, r.Host, Redact(r.RequestURI))

	if ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData); ok {
		logFields["aco_id"] = ad.ACOID
		logFields["token_id"] = ad.TokenID
		logFields["cms_id"] = ad.CMSID
	}

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Infoln("request started")

	return entry
}

type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

type ResourceTypeLogger struct {
	Repository models.JobKeyRepository
}

func (rl *ResourceTypeLogger) LogJobResourceType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		jobKey, err := rl.extractJobKey(r)
		if err != nil {
			logger := log.GetCtxLogger(r.Context())
			logger.Error(err)
			return
		}

		ctx, _ := log.SetCtxLogger(r.Context(), "resource_type", jobKey.ResourceType)
		r = r.WithContext(ctx)
	})
}

func (rl *ResourceTypeLogger) extractJobKey(r *http.Request) (*models.JobKey, error) {
	fileName := chi.URLParam(r, "fileName")
	jobID := chi.URLParam(r, "jobID")
	// Logging request for auditing

	jobIdInt, err := strconv.ParseUint(jobID, 10, 32)

	if err != nil {
		return nil, err
	}

	jobKey, err := rl.Repository.GetJobKey(r.Context(), uint(jobIdInt), strings.TrimSpace(fileName))

	return jobKey, err
}

func Redact(uri string) string {
	re := regexp.MustCompile(`Bearer%20([^&]+)(?:&|$)`)
	submatches := re.FindAllStringSubmatch(uri, -1)
	for _, match := range submatches {
		uri = strings.Replace(uri, match[1], "<redacted>", 1)
	}
	return uri
}

// NewCtxLogger adds new key value pair of {CtxLoggerKey: logrus.FieldLogger} to the requests context
func NewCtxLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logFields := logrus.Fields{}
		logFields["request_id"] = middleware.GetReqID(r.Context())
		if ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData); ok {
			logFields["cms_id"] = ad.CMSID
		}
		newLogEntry := &log.StructuredLoggerEntry{Logger: log.API.WithFields(logFields)}
		r = r.WithContext(context.WithValue(r.Context(), log.CtxLoggerKey, newLogEntry))
		next.ServeHTTP(w, r)
	})
}
