package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thiagohmm/producer/handlers"
)

func main() {
	handler := &handlers.HandlerCompra{}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("hello world"))
	// })
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Route("/v1", func(r chi.Router) {

		r.Post("/EnviaDadosCompra", handler.Salvar)
		r.Post("/EnviaDadosEstoque", handler.Salvar)
		r.Post("/EnviaDadosVendas", handler.Salvar)

	})
	http.ListenAndServe(":3009", r)
}
