package web

import (
	"github.com/CMSgov/bcda-app/bcda/auth"
	"net/http"
	"os"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/go-chi/chi"
)

func NewAPIRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(auth.ParseToken, logging.NewStructuredLogger(), HSTSHeader, ConnectionClose)

	// Serve up the swagger ui folder
	swagger_path := "./swaggerui"
	if _, err := os.Stat(swagger_path); os.IsNotExist(err) {
		swagger_path = "../swaggerui"
	}
	FileServer(r, "/api/v1/swagger", http.Dir(swagger_path))

	// Don't even load up the route in production
	if os.Getenv("DEPLOYMENT_TARGET") != "prod" {
		// Serve up the static site
		jekyll_path := "./_site"
		if _, err := os.Stat(jekyll_path); os.IsNotExist(err) {
			jekyll_path = "../_site"
		}
		FileServer(r, "/", http.Dir(jekyll_path))
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.With(auth.RequireTokenAuth, ValidateBulkRequestHeaders).Get(m.WrapHandler("/ExplanationOfBenefit/$export", bulkEOBRequest))
		if os.Getenv("ENABLE_PATIENT_EXPORT") == "true" {
			r.With(auth.RequireTokenAuth, ValidateBulkRequestHeaders).Get(m.WrapHandler("/Patient/$export", bulkPatientRequest))
		}
		if os.Getenv("ENABLE_COVERAGE_EXPORT") == "true" {
			r.With(auth.RequireTokenAuth, ValidateBulkRequestHeaders).Get(m.WrapHandler("/Coverage/$export", bulkCoverageRequest))
		}
		r.With(auth.RequireTokenAuth, auth.RequireTokenJobMatch).Get(m.WrapHandler("/jobs/{jobID}", jobStatus))
		r.Get(m.WrapHandler("/metadata", metadata))
	})
	r.Get(m.WrapHandler("/_version", getVersion))
	r.Get(m.WrapHandler("/_health", healthCheck))
	r.Get(m.WrapHandler("/_auth", getAuthInfo))
	return r
}

func NewAuthRouter() http.Handler {
	return auth.NewAuthRouter(logging.NewStructuredLogger(), HSTSHeader, ConnectionClose)
}

func NewDataRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(auth.ParseToken, logging.NewStructuredLogger(), HSTSHeader, ConnectionClose)
	r.With(auth.RequireTokenAuth, auth.RequireTokenJobMatch).
		Get(m.WrapHandler("/data/{jobID}/{fileName}", serveData))
	return r
}

func NewHTTPRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(ConnectionClose)
	r.With(logging.NewStructuredLogger()).Get(m.WrapHandler("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		url := "https://" + req.Host + req.URL.String()
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	})))
	return r
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
// stolen from https://github.com/go-chi/chi/blob/master/_examples/fileserver/main.go
func FileServer(r chi.Router, path string, root http.FileSystem) {
	m := monitoring.GetMonitor()
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(m.WrapHandler(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})))
}
