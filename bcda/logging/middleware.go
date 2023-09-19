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
	entry := &StructuredLoggerEntry{Logger: l.Logger}
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

func (l *StructuredLoggerEntry) Write(status int, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"resp_status": status, "resp_bytes_length": bytes,
		"resp_elapsed_ms": float64(elapsed.Nanoseconds()) / 1000000.0,
	})

	l.Logger.Infoln("request complete")
}

func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}

type ResourceTypeLogger struct {
	Repository models.JobKeyRepository
}

func (rl *ResourceTypeLogger) LogJobResourceType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		jobKey, err := rl.extractJobKey(r)
		if err != nil {
			log.API.Error(err)
			return
		}
		// Note: could split this out into a function for adding to the context log
		entry, ok := middleware.GetLogEntry(r).(*StructuredLoggerEntry)
		if !ok {
			log.API.Error("Incorrect type of logger used in request context")
			return
		}

		entry.Logger = entry.Logger.WithFields(logrus.Fields{
			"resource_type": jobKey.ResourceType,
		})
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

// type to create context.Contest key
type CtxLoggerKeyType string

// context.Context key to set/get logrus.FieldLogger value within request context
const CtxLoggerKey CtxLoggerKeyType = "ctxLogger"

// NewCtxLogger adds new key value pair of {CtxLoggerKey: logrus.FieldLogger} to the requests context
func NewCtxLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logFields := logrus.Fields{}
		logFields["request_id"] = middleware.GetReqID(r.Context())
		if ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData); ok {
			logFields["cms_id"] = ad.CMSID
		}
		newLogEntry := &StructuredLoggerEntry{Logger: log.API.WithFields(logFields)}
		r = r.WithContext(context.WithValue(r.Context(), CtxLoggerKey, newLogEntry))
		next.ServeHTTP(w, r)
	})
}

// Gets the logrus.FieldLogger from a context
func GetCtxLogger(ctx context.Context) logrus.FieldLogger {
	entry := ctx.Value(CtxLoggerKey).(*StructuredLoggerEntry)
	return entry.Logger
}

// Appends additional or creates new logrus.Fields to a logrus.FieldLogger within a context
func SetCtxLogger(ctx context.Context, key string, value interface{}) (context.Context, logrus.FieldLogger) {
	if entry, ok := ctx.Value(CtxLoggerKey).(*StructuredLoggerEntry); ok {
		entry.Logger = entry.Logger.WithField(key, value)
		nCtx := context.WithValue(ctx, CtxLoggerKey, entry)
		return nCtx, entry.Logger
	}

	var lggr logrus.Logger
	newLogEntry := &StructuredLoggerEntry{Logger: lggr.WithField(key, value)}
	nCtx := context.WithValue(ctx, CtxLoggerKey, newLogEntry)
	return nCtx, newLogEntry.Logger
}
