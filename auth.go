package main

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func parseJWT(r *http.Request) (*jwt.Token, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return nil, fmt.Errorf("no cookie found: %v", err)
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(appCtx.Config.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}
	return token, nil
}

func checkJWT(r *http.Request) bool {
	token, err := parseJWT(r)
	if err != nil || !token.Valid {
		appCtx.Logger.Warn("JWT validation failed: ", err)
		return false
	}
	return true
}
