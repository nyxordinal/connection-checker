package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
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
				http.Error(w, "Could not generate token", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    tokenString,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600,
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	http.ServeFile(w, r, "./static/login.html")
}

func rateLimitedHandler(rateLimiter *rate.Limiter, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiter.Allow() {
			appCtx.Logger.Warn("Rate limit reached due to too many requests")
			return
		}
		handler(w, r)
	}
}

func resetAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	token := r.Header.Get("Authorization")
	if token == "" || token != appCtx.Config.ResetToken {
		appCtx.Logger.Warn("Unauthorized attempt to reset alert status")
		return
	}

	appCtx.Mutex.Lock()
	appCtx.AlertSent = false
	appCtx.Mutex.Unlock()

	appCtx.Logger.Info("Alert status reset via HTTP endpoint")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := JsonResponse{
		Message: "Alert status reset successfully",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		appCtx.Logger.Error("Failed to encode JSON response: ", err)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status, lastEmailSent, err := appCtx.DB.getConnectionStatus()
	if err != nil {
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"connection_status": "unknown", "last_email_sent": ""})
			return
		}

		appCtx.Logger.Errorf("Failed to fetch connection status: %v", err)
		http.Error(w, "Failed to fetch connection status", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"connection_status": status,
		"last_email_sent":   lastEmailSent,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		http.Error(w, "Failed to fetch logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		appCtx.Logger.Errorf("Failed to encode logs: %v", err)
		http.Error(w, "Failed to encode logs", http.StatusInternalServerError)
	}
}
