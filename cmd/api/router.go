package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func initRouter() *chi.Mux {
	r := chi.NewRouter()

	registerMiddleware(r)
	registerRoutes(r)

	return r
}

func registerMiddleware(r *chi.Mux) *chi.Mux {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Heartbeat("/healthcheck"))

	return r
}

func registerRoutes(r *chi.Mux) *chi.Mux {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Materix!"))
	})

	return r
}
