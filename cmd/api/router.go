package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

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
		r.Route("/auth", func(r chi.Router) {
			r.Get("/github/callback", GithubOauthHandler)
			r.Get("/google/callback", GoogleOauthHandler)
		})
	})

	r.Mount("/api", apiRouter)

	return r
}

func GoogleOauthHandler(w http.ResponseWriter, r *http.Request) {
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	redirect_uri := "https://materix.up.railway.app/api/v1/auth/google/callback"

	baseAccessTokenUrl := "https://oauth2.googleapis.com/token"
	authTokenUrl := fmt.Sprintf("%s?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code", baseAccessTokenUrl, clientId, clientSecret, code, redirect_uri)

	if state != os.Getenv("OAUTH2_CALLBACK_STATE") {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid state"))
		return
	}

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

	tokenResponse := struct {
		Access_token  string `json:"access_token"`
		Scope         string `json:"scope"`
		Token_type    string `json:"token_type"`
		Expires_in    int    `json:"expires_in"`
		Refresh_token string `json:"refresh_token"`
		Id_token      string `json:"id_token"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	// TODO: Save user in database
	// TODO: Create and sign a JWT token
	// TODO: Send access token to user via json response
	data, err := json.MarshalIndent(tokenResponse, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(data))
}

func GithubOauthHandler(w http.ResponseWriter, r *http.Request) {
	clientId := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	redirect_uri := "https://materix.up.railway.app/api/v1/auth/github/callback"

	baseAccessTokenUrl := "https://github.com/login/oauth/access_token"
	authTokenUrl := fmt.Sprintf("%s?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code", baseAccessTokenUrl, clientId, clientSecret, code, redirect_uri)

	if state != os.Getenv("OAUTH2_CALLBACK_STATE") {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid state"))
		return
	}

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

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}

	err = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	// TODO: Save user in database
	// TODO: Create and sign a JWT token
	// TODO: Send access token to user via json response
	data, err := json.MarshalIndent(tokenResponse, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Internal server error: %s", err)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(data))
}
