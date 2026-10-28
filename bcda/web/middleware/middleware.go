package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"
)

// TODO: add body validation
type SharedExportBody struct {
	Since         string   `json:"since"`
	Version       string   `json:"version"`
	ResourceTypes []string `json:"resource_types"`
	PartnerID     string   `json:"partner_id"`
	DataTypes     []string `json:"data_types"`
	MBIs          []string `json:"mbis"`
	SinceDecoded  time.Time
}

// type to create context.Context key for the decoded SharedExportBody
type CtxSharedExportPayloadType struct{}

// context.Context key to get the SharedExportPayload from the request context

type ctxSharedExportKey struct{}

func WithSharedExportPayload(parent context.Context, body SharedExportBody) context.Context {
	return context.WithValue(parent, ctxSharedExportKey{}, body)
}

func FromContextSharedExportPayload(ctx context.Context) (SharedExportBody, bool) {
	u, ok := ctx.Value(ctxSharedExportKey{}).(SharedExportBody) // Type assertion
	return u, ok
}
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

func HasSharedExportAccess(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ad, ok := handleAuthData(r)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			rw, err := GetRespWriter(r.URL.Path)
			if err != nil {
				rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "unexpected API version")
				return
			}

			if !cfg.IsSharedExportModel(ad.CMSID) {
				ctx, _ := log.WriteWarnWithFields(
					r.Context(),
					fmt.Sprintf("failed to start shared job; CMSID %s does not have shared export access", ad.CMSID),
					logrus.Fields{"resp_status": http.StatusUnauthorized},
				)
				rw.OpOutcome(ctx, w, http.StatusUnauthorized, responseutils.InternalErr, "")
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func ValidateSharedExport(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rw, err := GetRespWriter(r.URL.Path)
		if err != nil {
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "unexpected API version")
			return
		}

		var payload SharedExportBody
		err = json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			log.API.Error("invalid JSON payload")
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "invalid JSON payload")
			return
		}
		defer r.Body.Close()

		if payload.Since == "" || len(payload.ResourceTypes) == 0 || payload.PartnerID == "" || len(payload.DataTypes) == 0 || len(payload.MBIs) == 0 {
			log.API.Error("missing required fields in payload")
			rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "missing required fields in payload")
			return
		}

		sinceDate, err := time.Parse(time.RFC3339Nano, payload.Since)
		if err != nil {
			log.API.Error("invalid _since parameter")
			rw.OpOutcome(r.Context(), w, http.StatusUnauthorized, responseutils.RequestErr, "")
			return
		}

		payload.SinceDecoded = sinceDate

		ctx := WithSharedExportPayload(r.Context(), payload)

		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func ACOEnabled(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ad, ok := handleAuthData(r)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			rw, err := GetRespWriter(r.URL.Path)
			if err != nil {
				rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "unexpected API version")
				return
			}

			if cfg.IsACODisabled(ad.CMSID) {
				ctx, _ := log.WriteWarnWithFields(
					r.Context(),
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

func V1V2DenyControl(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ad, ok := handleAuthData(r)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			rw, err := GetRespWriter(r.URL.Path)
			if err != nil {
				rw.OpOutcome(r.Context(), w, http.StatusBadRequest, responseutils.FormatErr, "unexpected API version")
				return
			}

			if cfg.IsACOV1V2DeniedAccess(ad.CMSID) {
				ctx, _ := log.WriteWarnWithFields(
					r.Context(),
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

func handleAuthData(r *http.Request) (auth.AuthData, bool) {
	ctx := r.Context()
	ad, ok := ctx.Value(auth.AuthDataContextKey).(auth.AuthData)
	if !ok {
		// We cannot get the correct FHIR response writer from here, so
		// return a non-FHIR-compliant HTTP response
		logger := log.GetCtxLogger(ctx)
		logger.WithField("resp_status", http.StatusInternalServerError).Error("AuthData should be set before calling this handler")

		return ad, false
	}

	return ad, true
}
