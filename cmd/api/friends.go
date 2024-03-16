package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/go-chi/chi"
)

func (app *application) getMyFriendsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	friends, err := app.models.Friends.GetFriendsFor(u.Id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	friendsJSON, err := json.MarshalIndent(friends, "", "\t")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(friendsJSON)
}

func (app *application) sendFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	var input struct {
		Id int `json:"id"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fRequest := &data.FriendRequest{
		SourceUserId:      u.Id,
		DestinationUserId: input.Id,
		Status:            "pending",
	}

	v := validator.New()
	if data.ValidateFriendPair(v, fRequest); !v.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		errors, _ := json.MarshalIndent(v.Errors, "", "\t")
		w.Write(errors)
		return
	}

	err = app.models.Friends.Insert(fRequest)
	if err != nil {
		switch err {
		case data.ErrDuplicateFriendRequest:
			http.Error(w, "Friend request already pending or accepted", http.StatusConflict)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Write([]byte(fmt.Sprintf("Friend request sent with id %d", fRequest.Id)))
}

func (app *application) acceptFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	fRequestId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fRequest, err := app.models.Friends.GetRequest(fRequestId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			http.Error(w, "Friend request not found", http.StatusNotFound)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	err = app.models.Friends.Accept(fRequest)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Friend request accepted"))
}

func (app *application) rejectFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	fRequestId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fRequest, err := app.models.Friends.GetRequest(fRequestId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			http.Error(w, "Friend request not found", http.StatusNotFound)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Ensure only the source or destination user can cancel/reject the request
	if u.Id != fRequest.DestinationUserId || u.Id != fRequest.SourceUserId {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = app.models.Friends.Delete(fRequest)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Friend request rejected"))
}

func (app *application) getSentFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	requests, err := app.models.Friends.GetSentFor(u.Id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	requestsJSON, err := json.MarshalIndent(requests, "", "\t")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(requestsJSON)
}

func (app *application) getReceivedFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	requests, err := app.models.Friends.GetReceivedFor(u.Id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	requestsJSON, err := json.MarshalIndent(requests, "", "\t")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(requestsJSON)
}

func (app *application) removeFriendHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	fId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	friendship, err := app.models.Friends.GetFriend(u.Id, fId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			http.Error(w, "Friend not found", http.StatusNotFound)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	err = app.models.Friends.Delete(friendship)
	if err != nil {
		switch err {
		case data.ErrEditConflict:
			http.Error(w, "Edit conflict", http.StatusConflict)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Write([]byte("Friend removed"))
}
