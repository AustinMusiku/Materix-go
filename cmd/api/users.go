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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request body"))
		return
	}

	// Create a new user and set details with input values
	u := data.User{
		Email:      input.Email,
		Name:       input.Name,
		Activated:  false,
		Avatar_url: "",
		Provider:   "email",
	}
	err = u.Password.Set(input.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	// Validate user details
	v := validator.New()
	if data.ValidateUser(v, &u); !v.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		errors, _ := json.MarshalIndent(v.Errors, "", "\t")
		w.Write([]byte(errors))
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
		w.WriteHeader(code)
		w.Write([]byte(msg))
		return
	}

	// TODO: Create and sign a JWT token
	// TODO: Send access token to user via json response
	user, err := json.MarshalIndent(&u, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	w.Write([]byte(user))
}

func (app *application) authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request body"))
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		errors, _ := json.MarshalIndent(v.Errors, "", "\t")
		w.Write([]byte(errors))
		return
	}

	u, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid email or password"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	if ok, err := u.Password.Compare(input.Password); !ok {
		if err == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid email or password"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}
		return
	}

	// TODO: Create and sign a JWT token
	// TODO: Send access token to user via json response

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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid state"))
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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	res, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}
	defer res.Body.Close()

	// Read response body
	token := struct {
		Access_token string `json:"access_token"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	// Extract user info from provider api
	userInfo, err := extractOauthUser(token.Access_token, oauthProvider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	// Check if user exists
	u, err := app.models.Users.GetByEmail(userInfo.Email)
	if err != nil && !errors.Is(err, data.ErrRecordNotFound) {
		// Create a new user
		u := data.User{
			Email:      userInfo.Email,
			Name:       userInfo.firstName + " " + userInfo.lastName,
			Activated:  true,
			Avatar_url: userInfo.Avatar_url,
			Provider:   oauthProvider,
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
			w.WriteHeader(code)
			w.Write([]byte(msg))
			return
		}
	}

	// TODO: Create and sign a JWT token
	// TODO: Send access token to user via json response
	user, err := json.MarshalIndent(u, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(user))
}

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid user id"))
		return
	}

	i, err := strconv.Atoi(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	u, err := app.models.Users.GetById(i)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("User not found"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}
		return
	}

	user, err := json.MarshalIndent(u, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	w.Write([]byte(user))
}

type oauthUserInfo struct {
	Email      string
	firstName  string
	lastName   string
	Provider   string
	Avatar_url string
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
		Avatar_url  string `json:"avatar_url"`
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
		userInfo.Avatar_url = body.Avatar_url
		userInfo.Email = body.Email
	} else if provider == "google" {
		userInfo.Email = body.Email
		userInfo.firstName = body.Given_name
		userInfo.lastName = body.Family_name
		userInfo.Avatar_url = body.Picture
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
