package main

import (
	"context"
	"errors"
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
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := bearer[7:]
		claims, err := data.ParseAccessToken(token)
		if err != nil {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		ok, err := claims.Verify()
		if err != nil || !ok {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		id, err := strconv.Atoi(claims.Sub)
		if err != nil {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		u, err := app.models.Users.GetById(id)
		if err != nil {
			switch err {
			case data.ErrRecordNotFound:
				app.notFoundResponse(w, r, errors.New("user not found"))
			default:
				app.serverErrorResponse(w, r, err)
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
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
