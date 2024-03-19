package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/go-chi/chi"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// Read request body into input struct
	var input struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Invalid request body"}, nil)
		return
	}

	// Create a new user and set details with input values
	u := data.User{
		Email:     input.Email,
		Name:      input.Name,
		Activated: false,
		AvatarUrl: "",
		Provider:  "email",
	}
	err = u.Password.Set(input.Password)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	// Validate user details
	v := validator.New()
	if data.ValidateUser(v, &u); !v.Valid() {
		errors, _ := json.MarshalIndent(v.Errors, "", "\t")
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": string(errors)}, nil)
	}

	// Save user in database
	err = app.models.Users.Insert(&u)
	if err != nil {
		var (
			code int
			msg  string
		)
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			code = http.StatusUnprocessableEntity
			msg = "A user with this email already exists"
		default:
			code = http.StatusInternalServerError
			msg = "Internal server error"
		}
		app.writeJSON(w, code, ResponseWrapper{"error": msg}, nil)
		return
	}

	tokens, err := data.NewTokenPair(u)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, ResponseWrapper{"tokens": tokens}, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (app *application) authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Invalid request body"}, nil)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		errors, _ := json.MarshalIndent(v.Errors, "", "\t")
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": string(errors)}, nil)
		return
	}

	u, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.writeJSON(w, http.StatusUnauthorized, ResponseWrapper{"error": "Invalid email or password"}, nil)
			return
		}
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	if ok, err := u.Password.Compare(input.Password); !ok {
		if err == nil {
			app.writeJSON(w, http.StatusUnauthorized, ResponseWrapper{"error": "Invalid email or password"}, nil)
		} else {
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	tokens, err := data.NewTokenPair(*u)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	jwts, err := json.MarshalIndent(tokens, "", "\t")
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"tokens": jwts}, nil)
}

func (app *application) oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	var clientId string
	var clientSecret string
	var baseAccessTokenUrl string
	var oauthProvider string

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	redirect_uri := "https://materix.up.railway.app/api/v1/auth/callback"

	// Verify state
	if state != os.Getenv("OAUTH2_CALLBACK_STATE") {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Invalid state"}, nil)
		return
	}

	// Assign provider value to oauthProvider after examining the request URL
	if strings.Contains(r.URL.String(), "google") {
		clientId = os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
		baseAccessTokenUrl = "https://oauth2.googleapis.com/token"
		oauthProvider = "google"
	} else {
		clientId = os.Getenv("GITHUB_CLIENT_ID")
		clientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
		baseAccessTokenUrl = "https://github.com/login/oauth/access_token"
		oauthProvider = "github"
	}

	// Exchange code for access token
	authTokenUrl := fmt.Sprintf("%s?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code", baseAccessTokenUrl, clientId, clientSecret, code, redirect_uri)
	req, err := http.NewRequest("POST", authTokenUrl, nil)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	res, err := client.Do(req)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}
	defer res.Body.Close()

	// Read response body
	token := struct {
		Access_token string `json:"access_token"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	// Extract user info from provider api
	userInfo, err := extractOauthUser(token.Access_token, oauthProvider)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	// Check if user exists
	u, err := app.models.Users.GetByEmail(userInfo.Email)
	if err != nil && !errors.Is(err, data.ErrRecordNotFound) {
		// Create a new user
		u = &data.User{
			Email:     userInfo.Email,
			Name:      userInfo.firstName + " " + userInfo.lastName,
			Activated: true,
			AvatarUrl: userInfo.AvatarUrl,
			Provider:  oauthProvider,
		}

		// Save user in database
		err = app.models.Users.Insert(u)
		if err != nil {
			var (
				code int
				msg  string
			)
			switch {
			case errors.Is(err, data.ErrDuplicateEmail):
				code = http.StatusUnprocessableEntity
				msg = "A user with this email already exists"
			default:
				code = http.StatusInternalServerError
				msg = "Internal server error"
			}
			app.writeJSON(w, code, ResponseWrapper{"error": msg}, nil)
			return
		}
	}

	tokens, err := data.NewTokenPair(*u)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"tokens": tokens}, nil)
}

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	i, err := strconv.Atoi(id)
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Bad request"}, nil)
		return
	}

	u, err := app.models.Users.GetById(i)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "User not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": u}, nil)
}

func (app *application) getMyUserHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	user, err := app.models.Users.GetById(u.Id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "User not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": user}, nil)
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	var input struct {
		Name      *string `json:"name"`
		Email     *string `json:"email"`
		AvatarUrl *string `json:"avatar"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, ResponseWrapper{"error": "Invalid request body"}, nil)
		return
	}

	u, err := app.models.Users.GetById(claims.Id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.writeJSON(w, http.StatusNotFound, ResponseWrapper{"error": "User not found"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	if input.Name != nil {
		u.Name = *input.Name
	}

	if input.Email != nil {
		u.Email = *input.Email
	}

	if input.AvatarUrl != nil {
		u.AvatarUrl = *input.AvatarUrl
	}

	v := validator.New()
	if data.ValidateUser(v, u); !v.Valid() {
		app.writeJSON(w, http.StatusUnprocessableEntity, ResponseWrapper{"error": v.Errors}, nil)
		return
	}

	err = app.models.Users.Update(u)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.writeJSON(w, http.StatusConflict, ResponseWrapper{"error": "Edit conflict"}, nil)
		default:
			app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		}
		return
	}

	app.writeJSON(w, http.StatusOK, ResponseWrapper{"user": u}, nil)
}

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	err := app.models.Users.Delete(u.Id)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, ResponseWrapper{"error": "Internal server error"}, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type oauthUserInfo struct {
	Email     string
	firstName string
	lastName  string
	Provider  string
	AvatarUrl string
}

func extractOauthUser(accessToken string, provider string) (oauthUserInfo, error) {
	var userInfo oauthUserInfo

	userEndpoint := "https://api.github.com/user"
	if provider == "google" {
		userEndpoint = "https://www.googleapis.com/oauth2/v3/userinfo"
	}

	// Request user profile info
	req, err := http.NewRequest("GET", userEndpoint, nil)
	if err != nil {
		return userInfo, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	res, err := client.Do(req)
	if err != nil {
		return userInfo, err
	}
	defer res.Body.Close()

	// Parse api response
	var body struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		AvatarUrl   string `json:"avatar_url"`
		Picture     string `json:"picture"`
		Given_name  string `json:"given_name"`
		Family_name string `json:"family_name"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return userInfo, err
	}

	if provider == "github" {
		if body.Email == "" {
			email, err := fetchGithubUserEmail(accessToken)
			if err != nil {
				return userInfo, err
			}
			body.Email = email
		}
		names := strings.Split(body.Name, " ")
		userInfo.firstName = names[0]
		userInfo.lastName = strings.Join(names[1:], " ")
		userInfo.AvatarUrl = body.AvatarUrl
		userInfo.Email = body.Email
	} else if provider == "google" {
		userInfo.Email = body.Email
		userInfo.firstName = body.Given_name
		userInfo.lastName = body.Family_name
		userInfo.AvatarUrl = body.Picture
	}
	userInfo.Provider = provider

	return userInfo, nil
}

func fetchGithubUserEmail(token string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	res, err := client.Do(req)
	if err != nil {
		return "", errors.New("failed to fetch user email")
	}
	defer res.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	err = json.NewDecoder(res.Body).Decode(&emails)
	if err != nil {
		return "", errors.New("failed to fetch user email")
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", errors.New("failed to fetch user email")
}
