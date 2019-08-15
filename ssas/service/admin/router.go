package admin

import (
	"fmt"
	"os"
	"time"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas/service"
)

var version = "latest"
var infoMap map[string][]string
var adminSigningKeyPath string
var server *service.Server

func init() {
	infoMap = make(map[string][]string)
	adminSigningKeyPath = os.Getenv("SSAS_ADMIN_SIGNING_KEY_PATH")
}

// Server creates an SSAS admin server
func Server() *service.Server {
	server = service.NewServer("admin", ":3004", version, infoMap, routes(), true, adminSigningKeyPath, 20*time.Minute)
	if server != nil {
		r, _ := server.ListRoutes()
		infoMap["banner"] = []string{fmt.Sprintf("%s server running on port %s", "admin", ":3004")}
		infoMap["routes"] = r
	}
	return server
}

func routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/group", createGroup)
	r.Get("/group", listGroups)
	r.Put("/group/{id}", updateGroup)
	r.Delete("/group/{id}", deleteGroup)
	r.Post("/system", createSystem)
	r.Put("/system/{systemID}/credentials", resetCredentials)
	r.Get("/system/{systemID}/key", getPublicKey)
	r.Delete("/system/{systemID}/credentials", deactivateSystemCredentials)
	r.Delete("/token/{tokenID}", revokeToken)
	return r
}
