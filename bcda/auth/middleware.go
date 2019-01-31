package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/responseutils"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

func RequireTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			log.Error("no token in header")
			respond(w, http.StatusUnauthorized)
			return
		}

		tokenString := strings.Split(authHeader, " ")[1]
		token, err := GetAuthProvider().DecodeJWT(tokenString)

		if err != nil {
			log.Error(err)
			respond(w, http.StatusUnauthorized)
			return
		}

		err = GetAuthProvider().ValidateJWT(tokenString)

		if err != nil {
			log.Error(err)
			respond(w, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "token", &token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireTokenACOMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value("token").(*jwt.Token)
		claims, err := ClaimsFromToken(token)
		if err != nil {
			log.Error(err)
			respond(w, http.StatusInternalServerError)
			return
		}

		re := regexp.MustCompile("/([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})(?:-error)?.ndjson")
		urlUUID := re.FindStringSubmatch(r.URL.String())[1]
		if uuid.Equal(uuid.Parse(claims["aco"].(string)), uuid.Parse(string(urlUUID))) {
			next.ServeHTTP(w, r)
		} else {
			log.Error(fmt.Errorf("token with ID: %v does not match ACO in request url", claims["id"]))
			respond(w, http.StatusUnauthorized)
			return
		}
	})
}

func ClaimsFromToken(token *jwt.Token) (jwt.MapClaims, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return jwt.MapClaims{}, errors.New("failed to determine token claims")
}

func respond(w http.ResponseWriter, status int) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
	responseutils.WriteError(oo, w, status)
}
