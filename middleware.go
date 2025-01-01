package main

import (
	"net/http"
)

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkJWT(r) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func authPageMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if checkJWT(r) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func apiAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkJWT(r) {
			constructResponse(w, http.StatusUnauthorized, "You are unauthorized", "Unauthorized")
			return
		}

		next(w, r)
	}
}
