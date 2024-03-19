package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/validator"
)

type oauthUserInfo struct {
	Email     string
	firstName string
	lastName  string
	Provider  string
	AvatarUrl string
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// Read request body into input struct
	var input struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.badRequestResponse(w, r, err)
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
		app.serverErrorResponse(w, r, err)
		return
	}

	// Validate user details
	v := validator.New()
	if data.ValidateUser(v, &u); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
	}

	// Save user in database
	err = app.models.Users.Insert(&u)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	tokens, err := data.NewTokenPair(u)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, ResponseWrapper{"tokens": tokens}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	u, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.invalidCredentialsResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if ok, err := u.Password.Compare(input.Password); !ok {
		if err == nil {
			app.invalidCredentialsResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	tokens, err := data.NewTokenPair(*u)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"tokens": tokens}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
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
		app.badRequestResponse(w, r, errors.New("invalid state"))
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
		app.serverErrorResponse(w, r, err)
		return
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	res, err := client.Do(req)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer res.Body.Close()

	// Read response body
	token := struct {
		Access_token string `json:"access_token"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Extract user info from provider api
	userInfo, err := extractOauthUser(token.Access_token, oauthProvider)
	if err != nil {
		app.serverErrorResponse(w, r, err)
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
			switch {
			case errors.Is(err, data.ErrDuplicateEmail):
				v := validator.New()
				v.AddError("email", "a user with this email address already exists")
				app.failedValidationResponse(w, r, v.Errors)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}
	}

	tokens, err := data.NewTokenPair(*u)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, ResponseWrapper{"tokens": tokens}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
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
