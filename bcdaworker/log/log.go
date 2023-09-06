package workerlog

import (
	"context"

	"github.com/sirupsen/logrus"
)

type logFieldsCtxKeyType string

const logFieldsCtxKey logFieldsCtxKeyType = "logFields"

func WithLogFields(ctx context.Context, fields logrus.Fields) context.Context {
	return context.WithValue(ctx, logFieldsCtxKey, fields)
}

func GetLogFields(ctx context.Context) logrus.Fields {
	logFields, ok := ctx.Value(logFieldsCtxKey).(logrus.Fields)
	if !ok {
		// Log this issue
		return nil
	}
	return logFields
}
