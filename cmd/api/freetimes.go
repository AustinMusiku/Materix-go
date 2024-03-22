package main

import (
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

	err := app.readJSON(w, r, &input)
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

	input := struct {
		data.Filters
		From time.Time
		To   time.Time
	}{}

	queryStrings := r.URL.Query()

	input.From = app.readDate(queryStrings, "from", "01-01-1970")
	input.To = app.readDate(queryStrings, "to", "01-01-2100")

	v := validator.New()
	input.Filters = data.Filters{
		Page:         app.readInt(queryStrings, "page", 1, v),
		PageSize:     app.readInt(queryStrings, "page_size", 20, v),
		Sort:         app.readString(queryStrings, "sort", "id"),
		SortSafelist: []string{"id", "start_time", "end_time", "created_at", "-id", "-start_time", "-end_time", "-created_at"},
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	freeTimes, meta, err := app.models.FreeTimes.GetAllFor(u.Id, input.Filters, input.From, input.To)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"meta": meta, "freetimes": freeTimes}, nil)
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
		app.badRequestResponse(w, r, err)
		return
	}

	var input struct {
		StartTime *time.Time `json:"start_time"`
		EndTime   *time.Time `json:"end_time"`
		Tags      *[]string  `json:"tags"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ft, err := app.models.FreeTimes.Get(fid)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("free time not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if ft.UserId != u.Id {
		app.notFoundResponse(w, r, errors.New("free time not found for user"))
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
		app.badRequestResponse(w, r, err)
		return
	}

	ft, err := app.models.FreeTimes.Get(fid)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("free time not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if ft.UserId != u.Id {
		app.notFoundResponse(w, r, errors.New("free time not found for user"))
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

	input := struct {
		data.Filters
		From time.Time
		To   time.Time
	}{}

	queryStrings := r.URL.Query()

	input.From = app.readDate(queryStrings, "from", "01-01-1970")
	input.To = app.readDate(queryStrings, "to", "01-01-2100")

	v := validator.New()
	input.Filters = data.Filters{
		Page:         app.readInt(queryStrings, "page", 1, v),
		PageSize:     app.readInt(queryStrings, "page_size", 50, v),
		Sort:         app.readString(queryStrings, "sort", "id"),
		SortSafelist: []string{"id", "start_time", "end_time", "created_at", "-id", "-start_time", "-end_time", "-created_at"},
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	freeTimes, meta, err := app.models.FreeTimes.GetAllForFriendsOf(u.Id, input.Filters, input.From, input.To)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"meta": meta, "freetimes": freeTimes}, nil)
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
		app.badRequestResponse(w, r, err)
		return
	}

	_, err = app.models.Friends.GetFriend(u.Id, friendId)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, r, errors.New("friend not found"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	input := struct {
		data.Filters
		From time.Time
		To   time.Time
	}{}

	queryStrings := r.URL.Query()
	input.From = app.readDate(queryStrings, "from", "01-01-1970")
	input.To = app.readDate(queryStrings, "to", "01-01-2100")

	v := validator.New()
	input.Filters = data.Filters{
		Page:         app.readInt(queryStrings, "page", 1, v),
		PageSize:     app.readInt(queryStrings, "page_size", 20, v),
		Sort:         app.readString(queryStrings, "sort", "id"),
		SortSafelist: []string{"id", "start_time", "end_time", "created_at", "-id", "-start_time", "-end_time", "-created_at"},
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	freeTimes, meta, err := app.models.FreeTimes.GetAllFor(friendId, data.Filters{}, time.Time{}, time.Time{})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"meta": meta, "freetimes": freeTimes}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
