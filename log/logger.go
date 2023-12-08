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
	API     logrus.FieldLogger
	Auth    logrus.FieldLogger
	BBAPI   logrus.FieldLogger
	Request logrus.FieldLogger
	SSAS    logrus.FieldLogger

	Worker   logrus.FieldLogger
	BBWorker logrus.FieldLogger
	Health   logrus.FieldLogger
)

func init() {
	setup()
}

// setup allows us to invoke it automatically (via init()) and in tests
// In tests, we want to set up the files/environment in a consistent manner
func setup() {
	env := conf.GetEnv("DEPLOYMENT_TARGET")

	API = logger(logrus.New(), conf.GetEnv("BCDA_ERROR_LOG"),
		"api", env)
	Auth = logger(logrus.New(), conf.GetEnv("AUTH_LOG"),
		"api", env)
	BBAPI = logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"),
		"api", env)
	Request = logger(logrus.New(), conf.GetEnv("BCDA_REQUEST_LOG"),
		"api", env)
	SSAS = logger(logrus.New(), conf.GetEnv("BCDA_SSAS_LOG"),
		"api", env)

	Worker = logger(logrus.New(), conf.GetEnv("BCDA_WORKER_ERROR_LOG"),
		"worker", env)
	BBWorker = logger(logrus.New(), conf.GetEnv("BCDA_BB_LOG"),
		"worker", env)
	Health = logger(logrus.New(), conf.GetEnv("WORKER_HEALTH_LOG"),
		"worker", env)
}

func logger(logger *logrus.Logger, outputFile string,
	application, environment string) logrus.FieldLogger {

	if outputFile != "" {
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

	return logger.WithFields(logrus.Fields{
		"application": application,
		"environment": environment,
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
