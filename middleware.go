package main

import "net/http"

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
			appCtx.Logger.Info("User is already authenticated")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		appCtx.Logger.Info("User is not authenticated")
		next(w, r)
	}
}

func apiAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkJWT(r) {
			return
		}
		next(w, r)
	}
}
