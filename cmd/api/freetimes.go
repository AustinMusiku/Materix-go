package main

import (
	"encoding/json"
	"errors"
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
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
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
		app.badRequestResponse(w, r, err)
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
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	insertedFreetime, err := app.models.FreeTimes.Insert(&ft, input.Viewers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, ResponseWrapper{"freetime": insertedFreetime}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getMyFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllFor(u.Id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetimes": freeTimes}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateFreeTimeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		app.badRequestResponse(w, r, errors.New("missing or invalid free time id"))
		return
	}

	fid, err := strconv.Atoi(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var input struct {
		StartTime *time.Time `json:"start_time"`
		EndTime   *time.Time `json:"end_time"`
		Tags      *[]string  `json:"tags"`
	}

	json.NewDecoder(r.Body).Decode(&input)

	ft, err := app.models.FreeTimes.Get(fid)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if ft.UserId != u.Id {
		app.notFoundResponse(w, r)
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
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	updatedFreetime, err := app.models.FreeTimes.Update(ft)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetime": updatedFreetime}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) removeFreeTimeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		app.badRequestResponse(w, r, errors.New("missing or invalid free time id"))
		return
	}

	fid, err := strconv.Atoi(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	ft, err := app.models.FreeTimes.Get(fid)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if ft.UserId != u.Id {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.FreeTimes.Delete(ft)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"message": "Free time removed"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getMyFriendsFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("context missing user value"))
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllForFriendsOf(u.Id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetimes": freeTimes}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getFriendFreeTimesHandler(w http.ResponseWriter, r *http.Request) {
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

	friendId, err := strconv.Atoi(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	_, err = app.models.Friends.GetFriend(u.Id, friendId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	freeTimes, err := app.models.FreeTimes.GetAllFor(friendId)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"freetimes": freeTimes}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
