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
	unsafeMode := os.Getenv("HTTP_ONLY") == "true"
	server = service.NewServer("admin", ":3004", version, infoMap, routes(), unsafeMode, adminSigningKeyPath, 20*time.Minute)
	if server != nil {
		r, _ := server.ListRoutes()
		infoMap["banner"] = []string{fmt.Sprintf("%s server running on port %s", "admin", ":3004")}
		infoMap["routes"] = r
	}
	return server
}

func routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(service.NewAPILogger(), service.ConnectionClose)
	r.With(requireBasicAuth).Post("/group", createGroup)
	r.With(requireBasicAuth).Get("/group", listGroups)
	r.With(requireBasicAuth).Put("/group/{id}", updateGroup)
	r.With(requireBasicAuth).Delete("/group/{id}", deleteGroup)
	r.With(requireBasicAuth).Post("/system", createSystem)
	r.With(requireBasicAuth).Put("/system/{systemID}/credentials", resetCredentials)
	r.With(requireBasicAuth).Get("/system/{systemID}/key", getPublicKey)
	r.With(requireBasicAuth).Delete("/system/{systemID}/credentials", deactivateSystemCredentials)
	r.With(requireBasicAuth).Delete("/token/{tokenID}", revokeToken)
	return r
}
