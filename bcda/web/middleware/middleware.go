package middleware

import (
	"fmt"
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/log"
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

func ACOEnabled(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
			if !ok {
				// We cannot get the correct FHIR response writer from here, so
				// return a non-FHIR-compliant HTTP response
				logger := log.GetCtxLogger(r.Context())
				logger.Error("AuthData should be set before calling this handler")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			rw, _ := getResponseWriterFromRequestPath(w, r)
			if rw == nil {
				return
			}

			if cfg.IsACODisabled(ad.CMSID) {
				logger := log.GetCtxLogger(r.Context())
				logger.Error(fmt.Sprintf("failed to complete request, CMSID %s is not enabled", ad.CMSID))
				rw.Exception(r.Context(), w, http.StatusUnauthorized, responseutils.InternalErr, "")
				return
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func V3AccessControl(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
			if !ok {
				// We cannot get the correct FHIR response writer from here, so
				// return a non-FHIR-compliant HTTP response
				logger := log.GetCtxLogger(r.Context())
				logger.Error("AuthData should be set before calling this handler")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			rw, _ := getResponseWriterFromRequestPath(w, r)
			if rw == nil {
				return
			}

			if !cfg.IsACOV3Enabled(ad.ACOID) {
				logger := log.GetCtxLogger(r.Context())
				logger.Error(fmt.Sprintf("failed to complete v3 request, ACOID %s does not have v3 access", ad.ACOID))
				rw.Exception(r.Context(), w, http.StatusForbidden, responseutils.UnauthorizedErr, "V3 access not enabled for this ACO")
				return
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
