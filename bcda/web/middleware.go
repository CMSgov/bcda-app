package web

import (
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
)

func ValidateBulkRequestHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header

		acceptHeader := h.Get("Accept")
		preferHeader := h.Get("Prefer")

		if acceptHeader == "" {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Structure, "", responseutils.FormatErr)
			oo.Issue[0].Diagnostics = "Accept header is required"
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		} else if acceptHeader != "application/fhir+json" {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Structure, "", responseutils.FormatErr)
			oo.Issue[0].Diagnostics = "application/fhir+json is the only supported response format"
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		if preferHeader == "" {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Structure, "", responseutils.FormatErr)
			oo.Issue[0].Diagnostics = "Prefer header is required"
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		} else if preferHeader != "respond-async" {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Structure, "", responseutils.FormatErr)
			oo.Issue[0].Diagnostics = "Only asynchronous responses are supported"
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
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
			w.Header().Set("X-Content-Type-Options", "no-sniff")
		}
		next.ServeHTTP(w, r)
	})
}
