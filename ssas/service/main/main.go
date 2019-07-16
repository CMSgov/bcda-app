package main

import (
	"flag"
	"log"
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
var unsafeMode bool

func init() {
	const usage = "start the service"
	flag.BoolVar(&startMeUp, "start", false, usage)
	flag.BoolVar(&startMeUp, "s", false, usage+" (shorthand)")
	unsafeMode = os.Getenv("HTTP_ONLY") == "true"
}

func main() {
	ssas.Logger.Info("Future home of the System-to-System Authentication Service")
	flag.Parse()
	if startMeUp {
		start()
	}
}

func hello() string {
	return "hello SSAS"
}

func start() {
	ssas.Logger.Infof("%s", "Starting ssas...")
	// if os.Getenv("DEBUG") == "true" {
		// autoMigrate()
	// }

	p := service.NewServer("public", ":3003", public.Version, public.InfoMap, public.Routes(), unsafeMode)
	p.LogRoutes()
	p.Serve()
	s := service.NewServer("admin", ":3004", admin.Version, admin.InfoMap, admin.Routes(), unsafeMode)
	s.LogRoutes()
	s.Serve()

	// Accepts and redirects HTTP requests to HTTPS.
	forwarder := &http.Server{
		Handler:      newForwardingRouter(),
		Addr:         ":3005",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	log.Fatal(forwarder.ListenAndServe())
}

func newForwardingRouter() http.Handler {
	r := chi.NewRouter()
	// todo middleware logging
	r.Get("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		url := "https://" + req.Host + req.URL.String()
		ssas.Logger.Infof("forwarding from %s to %s", req.Host+req.URL.String(), url)
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	}))
	return r
}
