package log

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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
	API = newFieldLogger("api", "api")
	Auth = newFieldLogger("api", "auth")
	BFDAPI = newFieldLogger("api", "bfd")
	Request = newFieldLogger("api", "request")
	SSAS = newFieldLogger("api", "ssas")

	Worker = newFieldLogger("worker", "worker")
	BFDWorker = newFieldLogger("worker", "bfd")
	Health = newFieldLogger("worker", "health")
}

// customize newFieldLogger and output to files
func newFieldLogger(application, logType string) logrus.FieldLogger {
	logger := newLogger()
	fields := defaultFields(application)
	fields["log_type"] = logType
	return logger.WithFields(fields)
}

func newLogger() *logrus.Logger {
	logger := logrus.New()
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
func NewSlogLogger(application string) *slog.Logger {
	logrusLogger := newLogger()
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

// Gets the logrus.FieldLogger from a context
func GetCtxLogger(ctx context.Context) logrus.FieldLogger {
	entry := ctx.Value(CtxLoggerKey)
	if entry != nil {
		return entry.(*StructuredLoggerEntry).Logger
	}
	return API
}

// Appends additional fields to our logger and sets it back into context
func SetLoggerFields(ctx context.Context, fields logrus.Fields) (context.Context, logrus.FieldLogger) {
	entry := ctx.Value(CtxLoggerKey).(*StructuredLoggerEntry)
	entry.Logger = entry.Logger.WithFields(fields)
	nCtx := context.WithValue(ctx, CtxLoggerKey, entry)

	return nCtx, entry.Logger
}

// Sets fields into logger, writes error entry, and sets logger back into context
func WriteErrorWithFields(ctx context.Context, msg string, fields logrus.Fields) (context.Context, logrus.FieldLogger) {
	logger := GetCtxLogger(ctx)
	logger = logger.WithFields(fields)
	logger.Error(msg)

	nCtx := context.WithValue(ctx, CtxLoggerKey, &StructuredLoggerEntry{Logger: logger})
	return nCtx, logger
}

// Sets fields into logger, writes warning entry, and sets logger back into context
func WriteWarnWithFields(ctx context.Context, msg string, fields logrus.Fields) (context.Context, logrus.FieldLogger) {
	logger := GetCtxLogger(ctx)
	logger = logger.WithFields(fields)
	logger.Warn(msg)

	nCtx := context.WithValue(ctx, CtxLoggerKey, &StructuredLoggerEntry{Logger: logger})
	return nCtx, logger
}

// Sets fields into logger, writes info entry, and sets logger back into context
func WriteInfoWithFields(ctx context.Context, msg string, fields logrus.Fields) (context.Context, logrus.FieldLogger) {
	logger := GetCtxLogger(ctx)
	logger = logger.WithFields(fields)
	logger.Info(msg)

	nCtx := context.WithValue(ctx, CtxLoggerKey, &StructuredLoggerEntry{Logger: logger})
	return nCtx, logger
}
