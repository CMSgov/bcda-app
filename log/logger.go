package log

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/sirupsen/logrus"
)

var (
	API     logrus.FieldLogger = defaultLogger("api-error")
	Auth    logrus.FieldLogger = defaultLogger("api-auth")
	BFDAPI  logrus.FieldLogger = defaultLogger("api-bfd")
	Request logrus.FieldLogger = defaultLogger("api-request")
	SSAS    logrus.FieldLogger = defaultLogger("ssas")

	Worker    logrus.FieldLogger = defaultLogger("worker")
	BFDWorker logrus.FieldLogger = defaultLogger("worker-bfd")
	Health    logrus.FieldLogger = defaultLogger("worker-health")
)

// setup global access to loggers, overwrite default logger
func SetupLoggers() {
	API = logger(logrus.New(), conf.GetEnv("BCDA_ERROR_LOG"), "api", "api-error")
	Auth = logger(logrus.New(), conf.GetEnv("AUTH_LOG"), "api", "api-auth")
	BFDAPI = logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"), "api", "api-bfd")
	Request = logger(logrus.New(), conf.GetEnv("BCDA_REQUEST_LOG"), "api", "api-request")
	SSAS = logger(logrus.New(), conf.GetEnv("BCDA_SSAS_LOG"), "api", "ssas")

	Worker = logger(logrus.New(), conf.GetEnv("BCDA_WORKER_ERROR_LOG"), "worker", "worker-error")
	BFDWorker = logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"), "worker", "worker-bfd")
	Health = logger(logrus.New(), conf.GetEnv("WORKER_HEALTH_LOG"), "worker", "worker-health")
}

// customize logger and output to files
func logger(logger *logrus.Logger, outputFile string, application string, logType string) logrus.FieldLogger {
	fields := logrus.Fields{
		"application": application,
		"environment": conf.GetEnv("DEPLOYMENT_TARGET"),
		"version":     constants.Version,
	}

	if conf.GetEnv("LOG_TO_STD_OUT") == "true" {
		fields["log_type"] = logType
	} else {
		if outputFile != "" {
			// #nosec G302 -- 0640 permissions required for Splunk ingestion
			if file, err := os.OpenFile(filepath.Clean(outputFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
				logger.SetOutput(file)
			} else {
				logger.Infof("Failed to open output file %s. Will use stderr. %s",
					outputFile, err.Error())
			}
		}
	}

	// Disable the HTML escape so we get the raw URLs
	logger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	logger.SetReportCaller(true)

	return logger.WithFields(fields)
}

// default logger, always available, outputs to stdout
func defaultLogger(logType string) logrus.FieldLogger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	logger.SetReportCaller(true)

	return logger.WithFields(logrus.Fields{
		"application": "default",
		"environment": conf.GetEnv("DEPLOYMENT_TARGET"),
		"log_type":    logType,
		"version":     constants.Version})
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
