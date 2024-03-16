package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
)

func (app *application) addFreeTimeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	input := struct {
		StartTime  time.Time `json:"start_time"`
		EndTime    time.Time `json:"end_time"`
		Tags       []string  `json:"tags"`
		Visibility string    `json:"visibility"`
		Viewers    []int     `json:"viewers"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ft := data.FreeTime{
		UserID:     u.Id,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Tags:       input.Tags,
		Visibility: input.Visibility,
	}

	v := validator.New()
	if valid := data.ValidateFreeTime(v, &ft); !valid {
		errors, _ := json.Marshal(v.Errors)
		http.Error(w, string(errors), http.StatusUnprocessableEntity)
		return
	}

	insertedFreetime, err := app.models.FreeTimes.Insert(&ft, input.Viewers)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(insertedFreetime)
}

func (app *application) getMyFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllFor(u.Id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(freeTimes)
}
