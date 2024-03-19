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
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
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
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	ft := data.FreeTime{
		UserId:     u.Id,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Tags:       input.Tags,
		Visibility: input.Visibility,
	}

	v := validator.New()
	if valid := data.ValidateFreeTime(v, &ft); !valid {
		errors, _ := json.Marshal(v.Errors)
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": "Invalid free time", "errors": errors}, nil)
		return
	}

	insertedFreetime, err := app.models.FreeTimes.Insert(&ft, input.Viewers)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusCreated, ResponseWrapper{"freetime": insertedFreetime}, nil)
}

func (app *application) getMyFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllFor(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetimes": freeTimes}, nil)
}

func (app *application) updateFreeTimeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
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
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "Free time not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	if ft.UserId != u.Id {
		app.writeJSON(w, http.StatusForbidden, ResponseWrapper{"error": "Forbidden"}, nil)
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
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": "Invalid free time", "errors": errors}, nil)
		return
	}

	updatedFreetime, err := app.models.FreeTimes.Update(ft)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetime": updatedFreetime}, nil)
}

func (app *application) removeFreeTimeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	ft, err := app.models.FreeTimes.Get(id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "Free time not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	if ft.UserId != u.Id {
		app.writeJSON(w, http.StatusForbidden, ResponseWrapper{"error": "Forbidden"}, nil)
		return
	}

	err = app.models.FreeTimes.Delete(ft)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Free time removed"}, nil)
}

func (app *application) getMyFriendsFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllForFriendsOf(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetimes": freeTimes}, nil)
}

func (app *application) getFriendFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	friendId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	_, err = app.models.Friends.GetFriend(u.Id, friendId)
	if err != nil {
		app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "Free time not found"}, nil)
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllFor(friendId)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetimes": freeTimes}, nil)
}
