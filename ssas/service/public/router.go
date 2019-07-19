package public

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
)

var Version = "latest"
var InfoMap map[string][]string

func init() {
	InfoMap = make(map[string][]string)
	InfoMap["public"] = []string{"token", "register", "authn/request", "authn/verify"}
}

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(service.NewAPILogger(), service.ConnectionClose)
	router.Get("/token", service.NYI)
	router.Post("/authn/request", RequestMultifactorChallenge)
	router.Post("/authn/verify", VerifyMultifactorResponse)
	router.With(fakeContext).Post("/register", RegisterSystem)

	return router
}

func fakeContext(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var	rd  ssas.AuthRegData
		if rd.GroupID = r.Header.Get("x-fake-token"); rd.GroupID == "" {
			service.GetLogEntry(r).Println("missing header x-fake-token; request will fail")
		}
		ctx := context.WithValue(r.Context(), "rd", rd)
		service.LogEntrySetField(r,"rd", rd)
		RegisterSystem(w, r.WithContext(ctx))
	})
}
