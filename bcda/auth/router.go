package auth

import (
"net/http"

"github.com/CMSgov/bcda-app/bcda/monitoring"
"github.com/go-chi/chi"
)

func NewAuthRouter(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(middlewares...)
	r.Post(m.WrapHandler("/auth/token", GetAuthToken))
	return r
}
