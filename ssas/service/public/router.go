package public

import (
	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas/service"
)

var Version = "latest"
var InfoMap map[string][]string

func init() {
	InfoMap = make(map[string][]string)
	InfoMap["public"] = []string{"token", "register"}
}

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/token", service.NYI)

	return router
}
