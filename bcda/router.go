package main

import (
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/go-chi/chi"
)

//NewRouter provides a router with all the required... routes
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.ParseToken, logging.NewStructuredLogger())
	// Serve up the swagger ui folder
	FileServer(r, "/api/v1/swagger", http.Dir("./swaggerui"))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello world!"))
		if err != nil {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.With(auth.RequireTokenAuth, ValidateBulkRequestHeaders).Get("/ExplanationOfBenefit/$export", bulkRequest)
		r.With(auth.RequireTokenAuth).Get("/jobs/{jobId}", jobStatus)
		r.Get("/metadata", metadata)
		r.Get("/_version", getVersion)
		if os.Getenv("DEBUG") == "true" {
			r.Get("/token", getToken)
			r.Get("/bb_metadata", blueButtonMetadata)
		}
	})
	r.With(auth.RequireTokenAuth, auth.RequireTokenACOMatch).Get("/data/{acoID}.ndjson", serveData)
	return r
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
// stolen from https://github.com/go-chi/chi/blob/master/_examples/fileserver/main.go
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}
