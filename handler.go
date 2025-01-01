package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/time/rate"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.ServeFile(w, r, "./static/index.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		appCtx.Logger.Info("Login attempt with username: ", username)

		if username == appCtx.Config.Username && password == appCtx.Config.Password {
			claims := &Claims{
				Username: username,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			secretKey := []byte(appCtx.Config.JWTSecret)
			tokenString, err := token.SignedString(secretKey)
			if err != nil {
				appCtx.Logger.Error("Could not generate token: ", err)
				constructResponse(w, http.StatusInternalServerError, "", "Login failed, please try again")
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    tokenString,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600,
			})
			return
		}

		constructResponse(w, http.StatusUnauthorized, "Unauthorized", "Invalid credentials")
		return
	}

	http.ServeFile(w, r, "./static/login.html")
}

func rateLimitedHandler(rateLimiter *rate.Limiter, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiter.Allow() {
			appCtx.Logger.Warn("Rate limit reached due to too many requests")
			constructResponse(w, http.StatusTooManyRequests, "Please slow down and try again in a moment.", "Too Many Requests")
			return
		}
		handler(w, r)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status, lastEmailSent, err := appCtx.DB.getConnectionStatus()
	if err != nil {
		if err == sql.ErrNoRows {
			constructResponseWithData(w, http.StatusOK, map[string]string{"connection_status": "unknown", "last_email_sent": ""})
			return
		}

		appCtx.Logger.Errorf("Failed to fetch connection status: %v", err)
		constructResponse(w, http.StatusInternalServerError, "", "Failed to fetch connection status")
		return
	}

	response := map[string]string{
		"connection_status": status,
		"last_email_sent":   lastEmailSent,
	}
	constructResponseWithData(w, http.StatusOK, response)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	page := 1
	perPage := 25
	if r.URL.Query().Get("page") != "" {
		page = atoi(r.URL.Query().Get("page"))
	}
	if r.URL.Query().Get("per_page") != "" {
		perPage = atoi(r.URL.Query().Get("per_page"))
	}

	logs, err := appCtx.DB.getConnectionLogs(page, perPage)
	if err != nil {
		appCtx.Logger.Errorf("Failed to fetch logs: %v", err)
		constructResponse(w, http.StatusInternalServerError, "", "Failed to fetch logs")
		return
	}

	constructResponseWithData(w, http.StatusOK, logs)
}

func resetAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	appCtx.Mutex.Lock()
	appCtx.AlertSent = false
	appCtx.Mutex.Unlock()

	appCtx.Logger.Info("Alert status reset triggered by user")
	constructResponse(w, http.StatusOK, "Alert status reset successfully", "")
}
