package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service/admin"
	"github.com/CMSgov/bcda-app/ssas/service/public"
)

var startMeUp bool
var migrateAndStart bool
var unsafeMode bool
var adminSigningKeyPath string

func init() {
	const usageStart = "start the service"
	flag.BoolVar(&startMeUp, "start", false, usageStart)
	flag.BoolVar(&startMeUp, "s", false, usageStart+" (shorthand)")
	const usageMigrate = "migrate the db and start the service"
	flag.BoolVar(&migrateAndStart, "migrate", false, usageMigrate)
	flag.BoolVar(&migrateAndStart, "m", false, usageMigrate+" (shorthand)")
	unsafeMode = os.Getenv("HTTP_ONLY") == "true"
}

func main() {
	ssas.Logger.Info("Future home of the System-to-System Authentication Service")
	flag.Parse()
	if migrateAndStart {
		if os.Getenv("DEBUG") == "true" {
			ssas.InitializeGroupModels()
			ssas.InitializeSystemModels()
		}
		start()
	}
	if startMeUp {
		start()
	}
}

func start() {
	ssas.Logger.Infof("%s", "Starting ssas...")

	ps := public.Server()
	if ps == nil {
		ssas.Logger.Error("unable to create public server")
		os.Exit(-1)
	}
	ps.LogRoutes()
	ps.Serve()

	as := admin.Server()
	if as == nil {
		ssas.Logger.Error("unable to create admin server")
		os.Exit(-1)
	}
	as.LogRoutes()
	as.Serve()

	// Accepts and redirects HTTP requests to HTTPS. Not sure we should do this.
	forwarder := &http.Server{
		Handler:      newForwardingRouter(),
		Addr:         ":3005",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ssas.Logger.Fatal(forwarder.ListenAndServe())
}

func newForwardingRouter() http.Handler {
	r := chi.NewRouter()
	// todo middleware logging
	r.Get("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO only forward requests for paths in our own host or resource server
		url := "https://" + req.Host + req.URL.String()
		ssas.Logger.Infof("forwarding from %s to %s", req.Host+req.URL.String(), url)
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	}))
	return r
}
