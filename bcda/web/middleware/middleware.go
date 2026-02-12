package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
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
			ctx := r.Context()
			ad, ok := ctx.Value(auth.AuthDataContextKey).(auth.AuthData)
			if !ok {
				// We cannot get the correct FHIR response writer from here, so
				// return a non-FHIR-compliant HTTP response
				logger := log.GetCtxLogger(ctx)
				logger.WithField("resp_status", http.StatusInternalServerError).Error("AuthData should be set before calling this handler")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			rw, _ := getResponseWriterFromRequestPath(w, r)
			if rw == nil {
				return
			}

			if cfg.IsACODisabled(ad.CMSID) {
				ctx, _ = log.WriteWarnWithFields(
					ctx,
					fmt.Sprintf("%s: Failed to complete request, CMSID %s is not enabled", responseutils.UnauthorizedErr, ad.CMSID),
					logrus.Fields{"resp_status": http.StatusUnauthorized},
				)
				rw.OpOutcome(ctx, w, http.StatusUnauthorized, responseutils.InternalErr, "")
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
			ctx := r.Context()
			ad, ok := ctx.Value(auth.AuthDataContextKey).(auth.AuthData)
			if !ok {
				// We cannot get the correct FHIR response writer from here, so
				// return a non-FHIR-compliant HTTP response
				logger := log.GetCtxLogger(ctx)
				logger.WithField("resp_status", http.StatusInternalServerError).Error("AuthData should be set before calling this handler")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			rw, _ := getResponseWriterFromRequestPath(w, r)
			if rw == nil {
				return
			}

			if !cfg.IsACOV3Enabled(ad.CMSID) {
				ctx, _ = log.WriteWarnWithFields(
					ctx,
					fmt.Sprintf("%s: Failed to begin v3 request, CMSID %s does not have v3 access", responseutils.UnauthorizedErr, ad.CMSID),
					logrus.Fields{"resp_status": http.StatusForbidden},
				)
				rw.OpOutcome(ctx, w, http.StatusForbidden, responseutils.UnauthorizedErr, "V3 access not enabled for this ACO")
				return
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func V1V2DenyControl(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ad, ok := ctx.Value(auth.AuthDataContextKey).(auth.AuthData)
			if !ok {
				// We cannot get the correct FHIR response writer from here, so
				// return a non-FHIR-compliant HTTP response
				logger := log.GetCtxLogger(ctx)
				logger.WithField("resp_status", http.StatusInternalServerError).Error("AuthData should be set before calling this handler")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			rw, _ := getResponseWriterFromRequestPath(w, r)
			if rw == nil {
				return
			}

			if isACOV1V2DeniedAccess(cfg, ad.CMSID) {
				ctx, _ = log.WriteWarnWithFields(
					ctx,
					fmt.Sprintf("%s: Failed to begin v1/v2 request, CMSID %s does not have v1/v2 access", responseutils.UnauthorizedErr, ad.CMSID),
					logrus.Fields{"resp_status": http.StatusForbidden},
				)
				rw.OpOutcome(ctx, w, http.StatusForbidden, responseutils.UnauthorizedErr, "v1 nor v2 access not enabled for this ACO")
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func isACOV1V2DeniedAccess(cfg *service.Config, ACOID string) bool {
	for _, str := range cfg.V1V2DenyRegexes {
		regex := regexp.MustCompile(str)
		if regex.MatchString(ACOID) {
			return true
		}
	}
	return false
}
