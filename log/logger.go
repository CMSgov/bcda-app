package log

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/conf"
	sloglogrus "github.com/samber/slog-logrus"
	"github.com/sirupsen/logrus"
)

var (
	API     logrus.FieldLogger = defaultFieldLogger("api")
	Auth    logrus.FieldLogger = defaultFieldLogger("auth")
	BFDAPI  logrus.FieldLogger = defaultFieldLogger("bfd")
	Request logrus.FieldLogger = defaultFieldLogger("request")
	SSAS    logrus.FieldLogger = defaultFieldLogger("ssas")

	Worker    logrus.FieldLogger = defaultFieldLogger("worker")
	BFDWorker logrus.FieldLogger = defaultFieldLogger("bfd")
	Health    logrus.FieldLogger = defaultFieldLogger("health")
)

// setup global access to loggers, overwrite default logger
func SetupLoggers() {
	API = newFieldLogger(conf.GetEnv("BCDA_ERROR_LOG"), "api", "api")
	Auth = newFieldLogger(conf.GetEnv("AUTH_LOG"), "api", "auth")
	BFDAPI = newFieldLogger(conf.GetEnv("BCDA_BB_LOG"), "api", "bfd")
	Request = newFieldLogger(conf.GetEnv("BCDA_REQUEST_LOG"), "api", "request")
	SSAS = newFieldLogger(conf.GetEnv("BCDA_SSAS_LOG"), "api", "ssas")

	Worker = newFieldLogger(conf.GetEnv("BCDA_WORKER_ERROR_LOG"), "worker", "worker")
	BFDWorker = newFieldLogger(conf.GetEnv("BCDA_BB_LOG"), "worker", "bfd")
	Health = newFieldLogger(conf.GetEnv("WORKER_HEALTH_LOG"), "worker", "health")
}

// customize newFieldLogger and output to files
func newFieldLogger(outputFile, application, logType string) logrus.FieldLogger {
	logger := newLogger(outputFile)
	fields := defaultFields(application)

	if conf.GetEnv("LOG_TO_STD_OUT") == "true" {
		fields["log_type"] = logType
	}
	return logger.WithFields(fields)
}

func newLogger(outputFile string) *logrus.Logger {
	logger := logrus.New()
	if conf.GetEnv("LOG_TO_STD_OUT") != "true" && outputFile != "" {
		// #nosec G302 -- 0640 permissions required for Splunk ingestion
		if file, err := os.OpenFile(filepath.Clean(outputFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
			logger.SetOutput(file)
		} else {
			logger.Infof("Failed to open output file %s. Will use stderr. %s",
				outputFile, err.Error())
		}
	}
	// Disable the HTML escape so we get the raw URLs
	logger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	logger.SetReportCaller(true)
	return logger
}

// default logger, always available, outputs to stdout
func defaultFieldLogger(logType string) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	logger.SetReportCaller(true)
	fields := defaultFields("default")
	fields["log_type"] = logType
	return logger.WithFields(fields)
}

func defaultFields(application string) logrus.Fields {
	return logrus.Fields{
		"application": application,
		"environment": conf.GetEnv("DEPLOYMENT_TARGET"),
		"source_app":  "bcda",
		"version":     constants.Version,
	}
}

// River requires a slog.Logger for logging, this function converts logrus to slog
func NewSlogLogger(outputFile, application string) *slog.Logger {
	logrusLogger := newLogger(outputFile)
	handler := sloglogrus.Option{Logger: logrusLogger}.NewLogrusHandler()
	return slogLoggerFromHandler(handler, application)
}

func slogLoggerFromHandler(handler slog.Handler, application string) *slog.Logger {
	return slog.New(handler).With(
		"application", application,
		"environment", conf.GetEnv("DEPLOYMENT_TARGET"),
		"source_app", "bcda",
		"version", constants.Version,
	)
}

// type to create context.Context key
type CtxLoggerKeyType string

// context.Context key to set/get logrus.FieldLogger value within request context
const CtxLoggerKey CtxLoggerKeyType = "ctxLogger"

type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

func (l *StructuredLoggerEntry) Write(status int, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"resp_status": status, "resp_bytes_length": bytes,
		"resp_elapsed_ms": float64(elapsed.Nanoseconds()) / 1000000.0,
	})

	if status >= 500 {
		l.Logger.Errorln("request complete")
	} else {
		l.Logger.Infoln("request complete")
	}
}

func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}

func NewStructuredLoggerEntry(logger logrus.FieldLogger, ctx context.Context) context.Context {
	newLogEntry := &StructuredLoggerEntry{Logger: logger.WithFields(logrus.Fields{})}
	ctx = context.WithValue(ctx, CtxLoggerKey, newLogEntry)
	return ctx
}

// Gets the logrus.StructuredLoggerEntry from a context
func GetCtxEntry(ctx context.Context) *StructuredLoggerEntry {
	entry := ctx.Value(CtxLoggerKey).(*StructuredLoggerEntry)
	return entry
}

// Gets the logrus.FieldLogger from a context
func GetCtxLogger(ctx context.Context) logrus.FieldLogger {
	entry := ctx.Value(CtxLoggerKey)
	if entry != nil {
		return entry.(*StructuredLoggerEntry).Logger
	}
	return API
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
