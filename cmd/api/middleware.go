package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/AustinMusiku/Materix-go/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := r.Header.Get("Authorization")
		if bearer == "" {
			ctx := context.WithValue(r.Context(), userContextKey, &data.User{})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if len(bearer) < 8 || strings.ToUpper(bearer[0:6]) != "BEARER" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := bearer[7:]
		claims, err := data.ParseAccessToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ok, err := claims.Verify()
		if err != nil || !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id, err := strconv.Atoi(claims.Sub)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		u, err := app.models.Users.GetById(id)
		if err != nil {
			switch err {
			case data.ErrRecordNotFound:
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			default:
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userContextKey).(*data.User)
		if !ok || u.CreatedAt == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
