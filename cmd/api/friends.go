package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/AustinMusiku/Materix-go/internal/data"
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

	err = app.models.Friends.Insert(u.Id, input.Id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Friend request sent"))
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
