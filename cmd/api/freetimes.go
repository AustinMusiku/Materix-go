package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/go-chi/chi"
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

func (app *application) updateFreeTimeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var input struct {
		StartTime *time.Time `json:"start_time"`
		EndTime   *time.Time `json:"end_time"`
		Tags      *[]string  `json:"tags"`
	}

	json.NewDecoder(r.Body).Decode(&input)

	ft, err := app.models.FreeTimes.Get(id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			http.Error(w, "Not found", http.StatusNotFound)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if ft.UserID != u.Id {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if input.StartTime != nil {
		ft.StartTime = *input.StartTime
	}

	if input.EndTime != nil {
		ft.EndTime = *input.EndTime
	}

	if input.Tags != nil {
		ft.Tags = *input.Tags
	}

	v := validator.New()
	if valid := data.ValidateFreeTime(v, ft); !valid {
		errors, _ := json.Marshal(v.Errors)
		http.Error(w, string(errors), http.StatusUnprocessableEntity)
		return
	}

	updatedFreetime, err := app.models.FreeTimes.Update(ft)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedFreetime)
}
