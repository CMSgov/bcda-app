package auth

import (
	"net/http"
	"os"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/go-chi/chi"
)

func NewAuthRouter(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(middlewares...)
	r.Post(m.WrapHandler("/auth/token", GetAuthToken))
	// TODO: remove conditional when new authentication implemented for administrative endpoints
	if os.Getenv("DEBUG") == "true" {
		r.Get(m.WrapHandler("/auth/group", GetAuthGroups))
		r.Post(m.WrapHandler("/auth/group", CreateAuthGroup))
		r.Put(m.WrapHandler("/auth/group", EditAuthGroup))
		r.Delete(m.WrapHandler("/auth/group", DeactivateAuthGroup))
	}
	return r
}
