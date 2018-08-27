package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
)

func ParseToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		authBackend := InitAuthBackend()

		var keyFunc jwt.Keyfunc = func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			} else {
				return authBackend.PublicKey, nil
			}
		}

		token, err := request.ParseFromRequest(r, request.OAuth2Extractor, keyFunc)

		if err == nil {
			ctx := context.WithValue(r.Context(), "token", token)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func RequireTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authBackend := InitAuthBackend()
		tokenValue := r.Context().Value("token")

		if tokenValue == nil {
			http.Error(w, http.StatusText(401), 401)
			return
		}

		token := tokenValue.(*jwt.Token)

		if token.Valid && !authBackend.IsBlacklisted(token) {
			ctx := context.WithValue(r.Context(), "token", token)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, http.StatusText(401), 401)
			return
		}
	})
}
