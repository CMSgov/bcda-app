package main

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/go-chi/chi"
)

//NewRouter provides a router with all the required... routes
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.ParseToken)
	r.Use(logging.NewStructuredLogger())
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello world!"))
		if err != nil {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.With(auth.RequireTokenAuth).With(ValidateBulkRequestHeaders).Get("/Patient/$export", bulkRequest)
		r.With(auth.RequireTokenAuth).Get("/jobs/{jobId}", jobStatus)

		if os.Getenv("DEBUG") == "true" {
			r.Get("/token", getToken)
			r.Get("/bb_metadata", blueButtonMetadata)
		}
	})
	r.Route("/data", func(r chi.Router) {
		r.With(auth.RequireTokenAuth).With(auth.RequireTokenACOMatch).Get("/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson", serveData)
	})
	return r
}
