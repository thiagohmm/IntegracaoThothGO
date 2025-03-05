package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thiagohmm/producer/handlers"
)

func main() {

	// Inicializa conexões
	handlers.InitRabbitMQ()
	handlers.InitRedis()

	handler := &handlers.HandlerAtena{}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handlers.HealthCheckHandler)

	r.Route("/v1", func(r chi.Router) {

		r.Post("/EnviaDadosCompra", handler.Salvar)
		r.Post("/EnviaDadosEstoque", handler.Salvar)
		r.Post("/EnviaDadosVendas", handler.Salvar)
		r.Get("/getStatusProcesso/{processo}", handler.PegarStatusProcesso)
		r.Get("/getAll/{processo}", handler.PegarTudo)
		r.Get("/requeue", handler.Reenqueue)
		r.Post("/filter", handler.Filtrar)
	})
	log.Println("Servidor iniciado com sucesso na porta 3009")
	http.ListenAndServe(":3009", r)
}
