package auth

import (
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/go-chi/chi/v5"
)

func NewAuthRouter(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(middlewares...)
	r.Post(m.WrapHandler("/auth/token", GetAuthToken))
	r.With(ParseToken, RequireTokenAuth, CheckBlacklist).Get(m.WrapHandler("/auth/welcome", Welcome))
	return r
}
