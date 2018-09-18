package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/pborman/uuid"
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

		if token.Valid {
			blacklisted, err := authBackend.IsBlacklisted(token)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if !blacklisted {
				ctx := context.WithValue(r.Context(), "token", token)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		http.Error(w, http.StatusText(401), 401)
	})
}

func RequireTokenACOMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenValue := r.Context().Value("token")

		if tokenValue == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if token, ok := tokenValue.(*jwt.Token); ok && token.Valid {
			claims, err := ClaimsFromToken(token)
			if err != nil {
				log.Error(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			aco, _ := claims["aco"].(string)

			re := regexp.MustCompile("/([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}).ndjson")
			urlUUID := re.FindStringSubmatch(r.URL.String())[1]

			if uuid.Equal(uuid.Parse(aco), uuid.Parse(string(urlUUID))) {
				next.ServeHTTP(w, r)
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
		}
	})
}

func ClaimsFromToken(token *jwt.Token) (jwt.MapClaims, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return jwt.MapClaims{}, errors.New("Error determining token claims")
}
