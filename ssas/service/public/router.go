package public

import (
	"os"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas/service"
)

var version = "latest"
var infoMap map[string][]string
var publicSigningKeyPath string
var server *service.Server

func init() {
	infoMap = make(map[string][]string)
	infoMap["public"] = []string{"token", "register", "authn/request", "authn/verify"}
	publicSigningKeyPath = os.Getenv("SSAS_PUBLIC_SIGNING_KEY_PATH")
}

func MakeServer() (*service.Server, error) {
	server = service.NewServer("public", ":3003", version, infoMap, routes(), true)
	// the signing key is separate from the [future] cert / private key used for https or tls or whatever
	if err := server.SetSigningKeys(publicSigningKeyPath); err != nil {
		return &service.Server{}, err
	}
	return server, nil
}

func routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(service.NewAPILogger(), service.ConnectionClose)
	router.Get("/token", token)
	router.Post("/authn", VerifyPassword)
	router.With(parseToken, requireMFATokenAuth).Post("/authn/request", RequestMultifactorChallenge)
	router.With(parseToken, requireMFATokenAuth).Post("/authn/verify", VerifyMultifactorResponse)
	router.With(parseToken, requireRegTokenAuth, readGroupID).Post("/register", RegisterSystem)
	router.With(parseToken, requireRegTokenAuth, readGroupID).Post("/reset", ResetSecret)

	return router
}

