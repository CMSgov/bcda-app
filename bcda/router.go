package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

//NewRouter provides a router with all the required... routes
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello world!"))
		if err != nil {
			log.Fatal(err)
		}
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/claims/{acoId}", bulkRequest)
		r.Get("/job/{jobId}", jobStatus)
	})
	return r
}
