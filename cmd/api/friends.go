package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/go-chi/chi"
)

func (app *application) getMyFriendsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	v := validator.New()
	queryStrings := r.URL.Query()

	filters := data.Filters{
		Page:         app.readInt(queryStrings, "page", 1, v),
		PageSize:     app.readInt(queryStrings, "page_size", 10, v),
		Sort:         app.readString(queryStrings, "sort", "id"),
		SortSafelist: []string{"id", "updated_at", "-id", "-updated_at"},
	}

	if data.ValidateFilters(v, filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	friends, meta, err := app.models.Friends.GetFriendsFor(u.Id, filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"meta": meta, "friends": friends}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) sendFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	var input struct {
		Id int `json:"id"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	fRequest := &data.FriendRequest{
		SourceUserId:      u.Id,
		DestinationUserId: input.Id,
		Status:            "pending",
	}

	v := validator.New()
	if data.ValidateFriendPair(v, fRequest); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Friends.Insert(fRequest)
	if err != nil {
		switch err {
		case data.ErrDuplicateFriendRequest:
			v.AddError("id", "friend request between users already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, ResponseWrapper{"message": "Friend request sent"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) acceptFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		app.badRequestResponse(w, r, errors.New("missing or invalid friend request id"))
		return
	}

	fRequestId, err := strconv.Atoi(id)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	fRequest, err := app.models.Friends.GetRequest(fRequestId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("friend request not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Ensure only the destination user can accept the request
	if u.Id != fRequest.DestinationUserId {
		app.notFoundResponse(w, r, errors.New("friend request not found for user"))
		return
	}

	err = app.models.Friends.Accept(fRequest)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Friend request accepted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) rejectFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		app.badRequestResponse(w, r, errors.New("missing or invalid friend request id"))
		return
	}

	fRequestId, err := strconv.Atoi(id)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	fRequest, err := app.models.Friends.GetRequest(fRequestId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("friend request not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Ensure only the source or destination user can cancel/reject the request
	if u.Id != fRequest.DestinationUserId || u.Id != fRequest.SourceUserId {
		app.notFoundResponse(w, r, errors.New("friend request not found for user"))
		return
	}

	err = app.models.Friends.Delete(fRequest)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Friend request rejected"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getSentFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	v := validator.New()
	queryStrings := r.URL.Query()

	filters := data.Filters{
		Page:         app.readInt(queryStrings, "page", 1, v),
		PageSize:     app.readInt(queryStrings, "page_size", 10, v),
		Sort:         app.readString(queryStrings, "sort", "id"),
		SortSafelist: []string{"id", "created_at", "-id", "-created_at"},
	}

	if data.ValidateFilters(v, filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	requests, meta, err := app.models.Friends.GetSentFor(u.Id, filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"meta": meta, "requests": requests}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getReceivedFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	v := validator.New()
	queryStrings := r.URL.Query()

	filters := data.Filters{
		Page:         app.readInt(queryStrings, "page", 1, v),
		PageSize:     app.readInt(queryStrings, "page_size", 10, v),
		Sort:         app.readString(queryStrings, "sort", "id"),
		SortSafelist: []string{"id", "created_at", "-id", "-created_at"},
	}

	if data.ValidateFilters(v, filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	requests, meta, err := app.models.Friends.GetReceivedFor(u.Id, filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"met": meta, "requests": requests}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) removeFriendHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		app.badRequestResponse(w, r, errors.New("missing or invalid friend id"))
		return
	}

	fId, err := strconv.Atoi(id)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	friendship, err := app.models.Friends.GetFriend(u.Id, fId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("friend not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.Friends.Delete(friendship)
	if err != nil {
		switch err {
		case data.ErrEditConflict:
			app.notFoundResponse(w, r, errors.New("friend not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Friend removed"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
