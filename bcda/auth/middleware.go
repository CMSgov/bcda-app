package auth

import (
	"context"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"net/http"
	"regexp"
	log "github.com/sirupsen/logrus"
	"strconv"
)

// Puts the decoded token and identity values into the request context. Decoded values have been
// verified to be tokens signed by our server and to have not expired. Additional authorization
// occurs in RequireTokenAuth(). Only auth code should look at the token claims; API code should
// rely on the values in AuthData. We do this to insulate API code from the differences among
// Provider tokens.
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
			log.Errorf("Unable to decode Authorization header value; %s", err)
			next.ServeHTTP(w, r)
			return
		}

		var ad AuthData
		if claims, ok := token.Claims.(*CommonClaims); ok && token.Valid {
			// okta token
			if claims.ClientID != "" && claims.Subject == claims.ClientID {
				var aco, err = GetACOByClientID(claims.ClientID)
				if err != nil {
					log.Errorf("no aco for clientID %s because %v", claims.ClientID, err)
					next.ServeHTTP(w, r)
					return
				}

				db := database.GetGORMDbConnection()
				defer database.Close(db)

				var user models.User
				if db.First(&user, "ACOID = ?", aco.UUID).RecordNotFound() {
					log.Errorf("no user for ACO with id of %v", aco.UUID)
					next.ServeHTTP(w, r)
					return
				}

				ad.TokenID = claims.Id
				ad.ACOID = aco.UUID.String()
				ad.UserID = user.UUID.String()
			} else {
				ad.TokenID = claims.UUID
				ad.ACOID = claims.ACOID
				ad.UserID = claims.Subject
			}
		}
		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "ad", ad)
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

func RequireTokenJobMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ad, ok := r.Context().Value("ad").(AuthData)
		if !ok {
			log.Error()
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Not_found)
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		jobID := chi.URLParam(r, "jobID")
		i, err := strconv.Atoi(jobID)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Not_found)
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		db := database.GetGORMDbConnection()
		defer database.Close(db)

		var job models.Job
		err = db.Find(&job, "id = ? and aco_id = ?", i, ad.ACOID).Error
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.Not_found)
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respond(w http.ResponseWriter, status int) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
	responseutils.WriteError(oo, w, status)
}
