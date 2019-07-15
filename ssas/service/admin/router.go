package admin

import (
	"net/http"

	"github.com/go-chi/chi"
)

func NewRouter(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Post("/group", createGroup)
	r.Put("/group/{id}", updateGroup)
	r.Post("/system", createSystem)
	return r
}
