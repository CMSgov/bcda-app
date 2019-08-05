package public

import (
	"fmt"
	"os"
	"time"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas/service"
)

var version = "latest"
var infoMap map[string][]string
var publicSigningKeyPath string
var server *service.Server

func init() {
	infoMap = make(map[string][]string)
	publicSigningKeyPath = os.Getenv("SSAS_PUBLIC_SIGNING_KEY_PATH")
}

func Server() (*service.Server) {
	server = service.NewServer("public", ":3003", version, infoMap, routes(), true, publicSigningKeyPath, 20 * time.Minute)
	if server != nil {
		r, _ := server.ListRoutes()
		infoMap["banner"] = []string{fmt.Sprintf("%s server running on port %s", "public", ":3003")}
		infoMap["routes"] = r
	}
	return server
}

func routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(service.NewAPILogger(), service.ConnectionClose)
	router.Get("/token", token)
	router.Post("/authn", VerifyPassword)
	router.With(parseToken, requireMFATokenAuth).Post("/authn/challenge", RequestMultifactorChallenge)
	router.With(parseToken, requireMFATokenAuth).Post("/authn/verify", VerifyMultifactorResponse)
	router.With(parseToken, requireRegTokenAuth, readGroupID).Post("/register", RegisterSystem)
	router.With(parseToken, requireRegTokenAuth, readGroupID).Post("/reset", ResetSecret)

	return router
}
