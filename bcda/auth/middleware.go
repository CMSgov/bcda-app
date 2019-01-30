package auth

import (
	"context"
	"errors"
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
		err := GetAuthProvider().ValidateJWT(tokenString)

		if err != nil {
			log.Error(err)
			respond(w, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "token", tokenString)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireTokenACOMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenValue := r.Context().Value("token")

		if tokenValue == nil {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
			responseutils.WriteError(oo, w, http.StatusUnauthorized)
			return
		}

		if token, ok := tokenValue.(*jwt.Token); ok && token.Valid {
			claims, err := ClaimsFromToken(token)
			if err != nil {
				log.Error(err)
				oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
				responseutils.WriteError(oo, w, http.StatusInternalServerError)
				return
			}

			aco, _ := claims["aco"].(string)

			re := regexp.MustCompile("/([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})(?:-error)?.ndjson")
			urlUUID := re.FindStringSubmatch(r.URL.String())[1]

			if uuid.Equal(uuid.Parse(aco), uuid.Parse(string(urlUUID))) {
				next.ServeHTTP(w, r)
			} else {
				log.Error("Token for incorrect ACO with ID: %v was rejected", token.Claims.(jwt.MapClaims)["id"])
				oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
				responseutils.WriteError(oo, w, http.StatusNotFound)
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

func respond(w http.ResponseWriter, status int) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
	responseutils.WriteError(oo, w, status)
}
