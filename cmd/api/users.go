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
		app.badRequestResponse(w, r, errors.New("missing or invalid user id"))
		return
	}

	i, err := strconv.Atoi(id)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	u, err := app.models.Users.GetById(i)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r, errors.New("user not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": u}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getMyUserHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	user, err := app.models.Users.GetById(u.Id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("user not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	var input struct {
		Name      *string `json:"name"`
		Email     *string `json:"email"`
		AvatarUrl *string `json:"avatar"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	u, err := app.models.Users.GetById(claims.Id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r, errors.New("user not found"))
		default:
			app.serverErrorResponse(w, r, err)
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
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Update(u)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": u}, nil)
}

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	err := app.models.Users.Delete(u.Id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"status": "success"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
