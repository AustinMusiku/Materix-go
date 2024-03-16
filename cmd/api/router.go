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
	r.Use(app.authenticate)

	r.Use(middleware.Heartbeat("/healthcheck"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Materix!"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/auth/callback", app.oauthCallbackHandler)
		r.Post("/auth/signup", app.registerUserHandler)
		r.Post("/auth/login", app.authenticateUserHandler)

		r.Get("/users/{id}", app.getUserHandler)
		// r.Get("/users/search", app.searchUsersHandler)

		r.Group(func(r chi.Router) {
			// require auth
			r.Use(app.requireAuthentication)

			r.Get("/users/me", app.getMyUserHandler)
			r.Patch("/users/me", app.updateUserHandler)
			r.Delete("/users/me", app.deleteUserHandler)

			r.Get("/friends", app.getMyFriendsHandler)
			// r.Delete("/friends/{id}", app.removeFriendHandler)

			r.Get("/friends/requests/sent", app.getSentFriendRequestsHandler)
			// r.Get("/friends/requests/received", app.getReceivedFriendRequestsHandler)
			r.Post("/friends/requests", app.sendFriendRequestHandler)
			r.Put("/friends/requests/{id}", app.acceptFriendRequestHandler)
			// r.Delete("/friends/requests/{id}", app.rejectFriendRequestHandler)

			// r.Get("/free", app.getMyFreeTimesHandler)
			// r.Post("/free", app.addFreeTimeHandler)
			// r.Patch("/free/{id}", app.updateFreeTimeHandler)
			// r.Delete("/free/{id}", app.removeFreeTimeHandler)

			// r.Get("/friends/free", app.getMyFriendsFreeTimesHandler)
			// r.Get("/friends/free/{id}", app.getFriendFreeTimesHandler)
		})
	})

	return r
}
