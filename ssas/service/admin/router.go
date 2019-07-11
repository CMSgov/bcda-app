package admin

import (
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/go-chi/chi"
)

func NewRouter(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	m := monitoring.GetMonitor()
	r.Use(middlewares...)
	r.Post(m.WrapHandler("/group", createGroup))
	r.Post("/system", createSystem)
	return r
}
