package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func initRouter() *chi.Mux {
	r := chi.NewRouter()

	registerMiddleware(r)
	registerRoutes(r)

	r = mountApiRouter(r)

	return r
}

func registerMiddleware(r *chi.Mux) *chi.Mux {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Heartbeat("/healthcheck"))

	return r
}

func registerRoutes(r *chi.Mux) *chi.Mux {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Materix!"))
	})

	return r
}

func mountApiRouter(r *chi.Mux) *chi.Mux {
	apiRouter := chi.NewRouter()

	apiRouter.Route("/v1", func(r chi.Router) {
		r.Get("/auth/callback", oauthCallbackHandler)
	})

	r.Mount("/api", apiRouter)

	return r
}

func oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
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

	// Assign provider value to oauthProvider variable by examining the request URL
	if strings.Contains(r.URL.String(), "google") {
		clientId = os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
		baseAccessTokenUrl = "https://oauth2.googleapis.com/token"
		oauthProvider = "google"
	} else if strings.Contains(r.URL.String(), "github") {
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	// Extract user info from id_token
	userInfo, err := extractOauthUser(string(body), oauthProvider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	// Create a new user
	user := data.User{
		Email:      userInfo.Email,
		FirstName:  userInfo.firstName,
		LastName:   userInfo.lastName,
		Activated:  true,
		Avatar_url: userInfo.Avatar_url,
		Provider:   oauthProvider,
		CreatedAt:  time.Now().Format(time.RFC3339),
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}

	// TODO: Save user in database
	// TODO: Create and sign a JWT token
	// TODO: Send access token to user via json response
	data, err := json.MarshalIndent(user, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(data))
}

type oauthUserInfo struct {
	Email      string
	firstName  string
	lastName   string
	Provider   string
	Avatar_url string
}

func extractOauthUser(token, provider string) (oauthUserInfo, error) {
	var userInfo oauthUserInfo

	claims := make(map[string]interface{})
	parts := strings.Split(token, ".")

	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return userInfo, err
	}

	err = json.Unmarshal(decoded, &claims)
	if err != nil {
		return userInfo, err
	}

	userInfo.Provider = provider
	if provider == "github" {
		names := strings.Split(claims["name"].(string), " ")
		userInfo.firstName = names[0]
		userInfo.lastName = strings.Join(names[1:], " ")
		userInfo.Avatar_url = claims["avatar_url"].(string)

		email, err := fetchGithubUserEmail(parts[0])
		if err != nil {
			return userInfo, err
		}
		userInfo.Email = email
	} else if provider == "google" {
		userInfo.Email = claims["email"].(string)
		userInfo.firstName = claims["given_name"].(string)
		userInfo.lastName = claims["family_name"].(string)
		userInfo.Avatar_url = claims["picture"].(string)
	}

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
