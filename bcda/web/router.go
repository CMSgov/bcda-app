package web

import (
	"net/http"
	"strings"

	v1 "github.com/CMSgov/bcda-app/bcda/api/v1"
	v2 "github.com/CMSgov/bcda-app/bcda/api/v2"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/go-chi/chi"
)

// Auth middleware checks that verifies that caller is authorized
var commonAuth = []func(http.Handler) http.Handler{
	auth.RequireTokenAuth,
	auth.CheckBlacklist}

func NewAPIRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(auth.ParseToken, logging.NewStructuredLogger(), SecurityHeader, ConnectionClose)

	// Serve up the swagger ui folder
	FileServer(r, "/api/v1/swagger", http.Dir("./swaggerui/v1"))

	if conf.GetEnv("DEPLOYMENT_TARGET") != "prod" {
		r.Get("/", userGuideRedirect)
		r.Get(`/{:(user_guide|encryption|decryption_walkthrough).html}`, userGuideRedirect)
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.With(append(commonAuth, ValidateBulkRequestHeaders)...).Get(m.WrapHandler("/Patient/$export", v1.BulkPatientRequest))
		r.With(append(commonAuth, ValidateBulkRequestHeaders)...).Get(m.WrapHandler("/Group/{groupId}/$export", v1.BulkGroupRequest))
		r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Get(m.WrapHandler("/jobs/{jobID}", v1.JobStatus))
		r.Get(m.WrapHandler("/metadata", v1.Metadata))
	})

	if utils.GetEnvBool("VERSION_2_ENDPOINT_ACTIVE", true) {
		FileServer(r, "/api/v2/swagger", http.Dir("./swaggerui/v2"))
		r.Route("/api/v2", func(r chi.Router) {
			r.With(append(commonAuth, ValidateBulkRequestHeaders)...).Get(m.WrapHandler("/Patient/$export", v2.BulkPatientRequest))
			r.With(append(commonAuth, ValidateBulkRequestHeaders)...).Get(m.WrapHandler("/Group/{groupId}/$export", v2.BulkGroupRequest))
			r.Get(m.WrapHandler("/metadata", v2.Metadata))
		})
	}

	r.Get(m.WrapHandler("/_version", v1.GetVersion))
	r.Get(m.WrapHandler("/_health", v1.HealthCheck))
	r.Get(m.WrapHandler("/_auth", v1.GetAuthInfo))
	return r
}

func NewAuthRouter() http.Handler {
	return auth.NewAuthRouter(logging.NewStructuredLogger(), SecurityHeader, ConnectionClose)
}

func NewDataRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(auth.ParseToken, logging.NewStructuredLogger(), SecurityHeader, ConnectionClose)
	r.With(append(commonAuth, auth.RequireTokenJobMatch)...).
		Get(m.WrapHandler("/data/{jobID}/{fileName}", v1.ServeData))
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

func userGuideRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, utils.FromEnv("USER_GUIDE_LOC", "https://bcda.cms.gov"), http.StatusMovedPermanently)
}
