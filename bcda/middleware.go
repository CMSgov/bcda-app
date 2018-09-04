package main

import (
	"net/http"
)

func ValidateBulkRequestHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header

		acceptHeader := h.Get("Accept")
		preferHeader := h.Get("Prefer")

		if acceptHeader == "" {
			http.Error(w, "Accept header is required", 400)
			return
		} else if acceptHeader != "application/fhir+json" {
			http.Error(w, "application/fhir+json is the only supported response format", 400)
			return
		}

		if preferHeader == "" {
			http.Error(w, "Prefer header is required", 400)
			return
		} else if preferHeader != "respond-async" {
			http.Error(w, "Only asynchronous responses are supported", 400)
			return
		}

		next.ServeHTTP(w, r)
	})
}
