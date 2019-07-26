package public

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
)

var version = "latest"
var infoMap map[string][]string
var publicSigningKeyPath string
var server *service.Server

func init() {
	infoMap = make(map[string][]string)
	infoMap["public"] = []string{"token", "register", "authn/request"}
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
	router.Post("/authn/request", RequestMultifactorChallenge)
	router.With(fakeContext).Post("/register", RegisterSystem)

	return router
}

func fakeContext(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rd ssas.AuthRegData
		if rd.GroupID = r.Header.Get("x-fake-token"); rd.GroupID == "" {
			service.GetLogEntry(r).Println("missing header x-fake-token; request will fail")
		}
		ctx := context.WithValue(r.Context(), "rd", rd)
		service.LogEntrySetField(r, "rd", rd)
		RegisterSystem(w, r.WithContext(ctx))
	})
}
