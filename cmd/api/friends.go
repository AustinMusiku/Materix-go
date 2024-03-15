package main

import (
	"encoding/json"
	"net/http"

	"github.com/AustinMusiku/Materix-go/internal/data"
)

func (app *application) getMyFriendsHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	friends, err := app.models.Users.GetFriendsFor(u.Id)
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
