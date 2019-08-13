package logging

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/servicemux"
)

// https://github.com/go-chi/chi/blob/master/_examples/logging/main.go

func NewStructuredLogger() func(next http.Handler) http.Handler {
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	filePath := os.Getenv("BCDA_REQUEST_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		logger.SetOutput(file)
	} else {
		logger.Info("Failed to open request log file; using default stderr")
	}
	return middleware.RequestLogger(&StructuredLogger{Logger: logger})
}

type StructuredLogger struct {
	Logger *logrus.Logger
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &StructuredLoggerEntry{Logger: logrus.NewEntry(l.Logger)}
	logFields := logrus.Fields{}

	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields["req_id"] = reqID
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

	logFields["uri"] = fmt.Sprintf("%s://%s%s", scheme, r.Host, Redact(r.RequestURI))

	if ad, ok := r.Context().Value("ad").(auth.AuthData); ok {
		logFields["aco_id"] = ad.ACOID
		logFields["user_id"] = ad.UserID
		logFields["token_id"] = ad.TokenID
	}

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Infoln("request started")

	return entry
}

type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

func (l *StructuredLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
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

func Redact(uri string) string {
	re := regexp.MustCompile(`Bearer%20([^&]+)(?:&|$)`)
	submatches := re.FindAllStringSubmatch(uri, -1)
	for _, match := range submatches {
		uri = strings.Replace(uri, match[1], "<redacted>", 1)
	}
	return uri
}
