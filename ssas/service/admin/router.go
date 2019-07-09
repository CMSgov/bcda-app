package admin

import (
	"github.com/go-chi/chi"
	"net/http"
)

func NewRouter(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Post("/system", createSystem)
	return r
}