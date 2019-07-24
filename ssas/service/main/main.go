package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
	"github.com/CMSgov/bcda-app/ssas/service/admin"
	"github.com/CMSgov/bcda-app/ssas/service/public"
)

var startMeUp bool
var migrateAndStart bool
var unsafeMode bool
var adminSigningKeyPath string
var publicSigningKeyPath string

func init() {
	const usageStart = "start the service"
	flag.BoolVar(&startMeUp, "start", false, usageStart)
	flag.BoolVar(&startMeUp, "s", false, usageStart + " (shorthand)")
	const usageMigrate = "migrate the db and start the service"
	flag.BoolVar(&migrateAndStart, "migrate", false, usageMigrate)
	flag.BoolVar(&migrateAndStart, "m", false, usageMigrate + " (shorthand)")
	unsafeMode = os.Getenv("HTTP_ONLY") == "true"
	adminSigningKeyPath = os.Getenv("SSAS_ADMIN_SIGNING_KEY_PATH")
	publicSigningKeyPath = os.Getenv("SSAS_PUBLIC_SIGNING_KEY_PATH")
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

	ps := service.NewServer("public", ":3003", public.Version, public.InfoMap, public.Routes(), unsafeMode)
	// the signing key is separate from the [future] cert / private key used for https or tls or whatever
	if err := ps.SetSigningKeys(publicSigningKeyPath); err != nil {
		ssas.Logger.Fatalf("unable to get signing key for public server because %s; can't start", err.Error())
	}

	ps.LogRoutes()
	ps.Serve()

	as := service.NewServer("admin", ":3004", admin.Version, admin.InfoMap, admin.Routes(), unsafeMode)
	// the signing key is separate from the [future] cert / private key used for https or tls or whatever
	if err := as.SetSigningKeys(adminSigningKeyPath); err != nil {
		ssas.Logger.Fatalf("unable to get signing key for admin server because %s; can't start", err.Error())
	}

	as.LogRoutes()
	as.Serve()

	// Accepts and redirects HTTP requests to HTTPS.
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
