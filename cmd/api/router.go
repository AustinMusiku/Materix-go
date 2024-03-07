package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func (app *application) initRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Heartbeat("/healthcheck"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Materix!"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/auth/callback", app.oauthCallbackHandler)
		r.Post("/auth/signup", app.registerUserHandler)

		r.Get("/users/{id}", app.getUserHandler)
		// r.Patch("/users/{id}", app.updateUserHandler)
		// r.Delete("/users/{id}", app.deleteUserHandler)
		// r.Get("/users/search", app.searchUsersHandler)
		// r.Get("/users", app.listUsersHandler)
	})

	return r
}
