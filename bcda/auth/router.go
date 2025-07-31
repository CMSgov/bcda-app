package auth

import (
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/go-chi/chi/v5"
)

func NewAuthRouter(provider Provider, middlewares ...func(http.Handler) http.Handler) http.Handler {
	baseApi := NewBaseApi(provider)
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(middlewares...)
	r.Post(m.WrapHandler("/auth/token", baseApi.GetAuthToken))
	r.With(ParseToken, RequireTokenAuth, CheckBlacklist).Get(m.WrapHandler("/auth/welcome", baseApi.Welcome))
	return r
}
