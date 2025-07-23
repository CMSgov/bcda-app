package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	v1 "github.com/CMSgov/bcda-app/bcda/api/v1"
	v2 "github.com/CMSgov/bcda-app/bcda/api/v2"
	v3 "github.com/CMSgov/bcda-app/bcda/api/v3"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/conf"
	appMiddleware "github.com/CMSgov/bcda-app/middleware"
	"github.com/go-chi/chi/v5"
	gcmw "github.com/go-chi/chi/v5/middleware"
)

// Auth middleware checks that verifies that caller is authorized
var commonAuth = []func(http.Handler) http.Handler{
	auth.RequireTokenAuth,
	auth.CheckBlacklist}

func NewAPIRouter(connection *sql.DB) http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(gcmw.RequestID, appMiddleware.NewTransactionID, auth.ParseToken, logging.NewStructuredLogger(), middleware.SecurityHeader, middleware.ConnectionClose, logging.NewCtxLogger)

	// Serve up the swagger ui folder
	FileServer(r, "/api/v1/swagger", http.Dir("./swaggerui/v1"))

	cfg, err := service.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("could not load service config file: %w", err))
	}

	var requestValidators = []func(http.Handler) http.Handler{
		middleware.ACOEnabled(cfg), middleware.ValidateRequestURL, middleware.ValidateRequestHeaders, middleware.CheckConcurrentJobs(cfg),
	}
	nonExportRequestValidators := []func(http.Handler) http.Handler{
		middleware.ACOEnabled(cfg), middleware.ValidateRequestURL, middleware.ValidateRequestHeaders,
	}

	if conf.GetEnv("DEPLOYMENT_TARGET") != "prod" {
		r.Get("/", userGuideRedirect)
		r.Get(`/{:(user_guide|encryption|decryption_walkthrough).html}`, userGuideRedirect)
	}

	r.Route("/api/v1", func(r chi.Router) {
		apiV1 := v1.NewApiV1(connection)
		r.With(append(commonAuth, requestValidators...)...).Get(m.WrapHandler("/Patient/$export", apiV1.BulkPatientRequest))
		r.With(append(commonAuth, requestValidators...)...).Get(m.WrapHandler("/Group/{groupId}/$export", apiV1.BulkGroupRequest))
		r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Get(m.WrapHandler(constants.JOBIDPath, apiV1.JobStatus))
		r.With(append(commonAuth, nonExportRequestValidators...)...).Get(m.WrapHandler("/jobs", apiV1.JobsStatus))
		r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Delete(m.WrapHandler(constants.JOBIDPath, apiV1.DeleteJob))
		r.With(commonAuth...).Get(m.WrapHandler("/attribution_status", apiV1.AttributionStatus))
		r.Get(m.WrapHandler("/metadata", v1.Metadata))
	})

	if utils.GetEnvBool("VERSION_2_ENDPOINT_ACTIVE", true) {
		FileServer(r, "/api/v2/swagger", http.Dir("./swaggerui/v2"))
		apiV2 := v2.NewApiV2(connection)
		r.Route("/api/v2", func(r chi.Router) {
			r.With(append(commonAuth, requestValidators...)...).Get(m.WrapHandler("/Patient/$export", apiV2.BulkPatientRequest))
			r.With(append(commonAuth, requestValidators...)...).Get(m.WrapHandler("/Group/{groupId}/$export", apiV2.BulkGroupRequest))
			r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Get(m.WrapHandler(constants.JOBIDPath, apiV2.JobStatus))
			r.With(append(commonAuth, nonExportRequestValidators...)...).Get(m.WrapHandler("/jobs", apiV2.JobsStatus))
			r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Delete(m.WrapHandler(constants.JOBIDPath, apiV2.DeleteJob))
			r.With(commonAuth...).Get(m.WrapHandler("/attribution_status", apiV2.AttributionStatus))
			r.Get(m.WrapHandler("/metadata", apiV2.Metadata))
		})
	}

	if utils.GetEnvBool("VERSION_3_ENDPOINT_ACTIVE", true) {
		apiV3 := v3.NewApiV3(connection)
		r.Route("/api/demo", func(r chi.Router) {
			r.With(append(commonAuth, requestValidators...)...).Get(m.WrapHandler("/Patient/$export", apiV3.BulkPatientRequest))
			r.With(append(commonAuth, requestValidators...)...).Get(m.WrapHandler("/Group/{groupId}/$export", apiV3.BulkGroupRequest))
			r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Get(m.WrapHandler(constants.JOBIDPath, apiV3.JobStatus))
			r.With(append(commonAuth, nonExportRequestValidators...)...).Get(m.WrapHandler("/jobs", apiV3.JobsStatus))
			r.With(append(commonAuth, auth.RequireTokenJobMatch)...).Delete(m.WrapHandler(constants.JOBIDPath, apiV3.DeleteJob))
			r.With(commonAuth...).Get(m.WrapHandler("/attribution_status", apiV3.AttributionStatus))
			r.Get(m.WrapHandler("/metadata", apiV3.Metadata))
		})
	}

	r.Get(m.WrapHandler("/_version", v1.GetVersion))
	r.Get(m.WrapHandler("/_health", v1.HealthCheck))
	r.Get(m.WrapHandler("/_auth", v1.GetAuthInfo))
	return r
}

func NewAuthRouter() http.Handler {
	return auth.NewAuthRouter(gcmw.RequestID, appMiddleware.NewTransactionID, logging.NewStructuredLogger(), middleware.SecurityHeader, middleware.ConnectionClose, logging.NewCtxLogger)
}

func NewDataRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	resourceTypeLogger := &logging.ResourceTypeLogger{
		Repository: postgres.NewRepository(database.Connection),
	}
	r.Use(auth.ParseToken, gcmw.RequestID, appMiddleware.NewTransactionID, logging.NewStructuredLogger(), middleware.SecurityHeader, middleware.ConnectionClose, logging.NewCtxLogger)
	r.With(append(
		commonAuth,
		auth.RequireTokenJobMatch,
		resourceTypeLogger.LogJobResourceType,
	)...).Get(m.WrapHandler("/data/{jobID}/{fileName}", v1.ServeData))
	return r
}

func NewHTTPRouter() http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(gcmw.RequestID, middleware.ConnectionClose, appMiddleware.NewTransactionID, logging.NewCtxLogger)
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
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
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
