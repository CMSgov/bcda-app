package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

func ParseToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		authRegexp := regexp.MustCompile(`^Bearer (\S+)$`)
		authSubmatches := authRegexp.FindStringSubmatch(authHeader)
		if len(authSubmatches) < 2 {
			log.Warn("Invalid Authorization header value")
			next.ServeHTTP(w, r)
			return
		}

		tokenString := authSubmatches[1]
		token, err := GetProvider().DecodeJWT(tokenString)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), "token", token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value("token")
		if token == nil {
			log.Error("No token found")
			respond(w, http.StatusUnauthorized)
			return
		}

		if token, ok := token.(*jwt.Token); ok {
			err := GetProvider().ValidateJWT(token.Raw)
			if err != nil {
				log.Error(err)
				respond(w, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		}
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

func RequireTokenJobMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobId")
		token := r.Context().Value("token").(*jwt.Token)

		db := database.GetGORMDbConnection()
		defer database.Close(db)

		i, err := strconv.Atoi(jobID)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Processing)
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		claims, err := ClaimsFromToken(token)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		acoId := claims["aco"].(string)

		var job models.Job
		err = db.Find(&job, "id = ? and aco_id = ?", i, acoId).Error
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.DbErr)
			responseutils.WriteError(oo, w, http.StatusNotFound)
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
