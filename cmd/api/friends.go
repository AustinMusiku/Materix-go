package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/go-chi/chi"
)

func (app *application) getMyFriendsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	friends, err := app.models.Friends.GetFriendsFor(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"friends": friends}, nil)
}

func (app *application) sendFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	var input struct {
		Id int `json:"id"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	fRequest := &data.FriendRequest{
		SourceUserId:      u.Id,
		DestinationUserId: input.Id,
		Status:            "pending",
	}

	v := validator.New()
	if data.ValidateFriendPair(v, fRequest); !v.Valid() {
		errors, _ := json.MarshalIndent(v.Errors, "", "\t")
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": "Invalid friend request", "errors": errors}, nil)
		return
	}

	err = app.models.Friends.Insert(fRequest)
	if err != nil {
		switch err {
		case data.ErrDuplicateFriendRequest:
			app.writeJSON(w, http.StatusConflict, ResponseWrapper{"error": "Friend request already sent"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusCreated, ResponseWrapper{"message": "Friend request sent"}, nil)
}

func (app *application) acceptFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	fRequestId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	fRequest, err := app.models.Friends.GetRequest(fRequestId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "Friend request not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	err = app.models.Friends.Accept(fRequest)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Friend request accepted"}, nil)
}

func (app *application) rejectFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	fRequestId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	fRequest, err := app.models.Friends.GetRequest(fRequestId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "Friend request not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	// Ensure only the source or destination user can cancel/reject the request
	if u.Id != fRequest.DestinationUserId || u.Id != fRequest.SourceUserId {
		app.writeJSON(w, http.StatusForbidden, ResponseWrapper{"error": "Forbidden"}, nil)
		return
	}

	err = app.models.Friends.Delete(fRequest)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Friend request rejected"}, nil)
}

func (app *application) getSentFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	requests, err := app.models.Friends.GetSentFor(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"requests": requests}, nil)
}

func (app *application) getReceivedFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	requests, err := app.models.Friends.GetReceivedFor(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"requests": requests}, nil)
}

func (app *application) removeFriendHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	fId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	friendship, err := app.models.Friends.GetFriend(u.Id, fId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "Friend not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	err = app.models.Friends.Delete(friendship)
	if err != nil {
		switch err {
		case data.ErrEditConflict:
			app.writeJSON(w, http.StatusConflict, ResponseWrapper{"error": "Friend not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Friend removed"}, nil)
}
