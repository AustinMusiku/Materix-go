package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/go-chi/chi"
)

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	i, err := strconv.Atoi(id)
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	u, err := app.models.Users.GetById(i)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "User not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": u}, nil)
}

func (app *application) getMyUserHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	user, err := app.models.Users.GetById(u.Id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "User not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": user}, nil)
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	var input struct {
		Name      *string `json:"name"`
		Email     *string `json:"email"`
		AvatarUrl *string `json:"avatar"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Invalid request body"}, nil)
		return
	}

	u, err := app.models.Users.GetById(claims.Id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "User not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	if input.Name != nil {
		u.Name = *input.Name
	}

	if input.Email != nil {
		u.Email = *input.Email
	}

	if input.AvatarUrl != nil {
		u.AvatarUrl = *input.AvatarUrl
	}

	v := validator.New()
	if data.ValidateUser(v, u); !v.Valid() {
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": v.Errors}, nil)
		return
	}

	err = app.models.Users.Update(u)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.writeJSON(w, http.StatusConflict, ResponseWrapper{"error": "Edit conflict"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": u}, nil)
}

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	err := app.models.Users.Delete(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
