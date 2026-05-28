package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewAuthRouter(provider Provider, middlewares ...func(http.Handler) http.Handler) http.Handler {
	baseApi := NewBaseApi(provider)
	r := chi.NewRouter()
	am := NewAuthMiddleware(provider)
	r.Use(middlewares...)

	r.Post("/auth/token", baseApi.GetAuthToken)
	r.With(am.ParseToken, RequireTokenAuth, CheckBlacklist).Get("/auth/welcome", baseApi.Welcome)

	return r
}
