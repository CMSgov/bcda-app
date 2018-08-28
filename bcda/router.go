package main

import (
	"log"
	"net/http"
	"os"

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
			log.Fatal(err)
		}
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.With(auth.RequireTokenAuth).Get("/Patient/$export", bulkRequest)
		r.With(auth.RequireTokenAuth).Get("/jobs/{jobId}", jobStatus)

		if os.Getenv("DEBUG") == "true" {
			r.Get("/token", getToken)
		}
	})
	return r
}
