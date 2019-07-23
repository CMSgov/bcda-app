package admin

import (
	"github.com/go-chi/chi"
)

var Version = "latest"
var InfoMap = map[string][]string{}

func init() {
	InfoMap = make(map[string][]string)
	InfoMap["admin"] = []string{"system", "group"}
}

func Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/group", createGroup)
	r.Post("/system", createSystem)
	r.Put("/system/{systemID}/credentials", resetCredentials)
	r.Get("/system/{systemID}/key", getPublicKey)
	r.Delete("/system/{systemID}/credentials", deactivateSystemCredentials)
	return r
}
