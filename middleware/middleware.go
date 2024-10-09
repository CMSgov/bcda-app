package middleware

import (
	"context"
	"net/http"

	"github.com/pborman/uuid"
)

// type to create context.Context key
type CtxTransactionKeyType string

// context.Context key to get the transaction ID from the request context
const CtxTransactionKey CtxTransactionKeyType = "ctxTransaction"

// Adds a transaction ID to the request context
func NewTransactionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), CtxTransactionKey, uuid.New()))
		next.ServeHTTP(w, r)
	})
}
