package auth

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
)

// Use context keys for storing/retrieving data in the http Context
type contextKey struct {
	name string
}

var (
	TokenContextKey    = &contextKey{"token"}
	AuthDataContextKey = &contextKey{"ad"}
)

// ParseToken puts the decoded token and AuthData value into the request context. Decoded values come from
// tokens verified by our provider as correct and unexpired. Tokens may be presented in requests to
// unauthenticated endpoints (mostly swagger?). We still want to extract the token data for logging purposes,
// even when we don't use it for authorization. Authorization for protected endpoints occurs in RequireTokenAuth().
// Only auth code should look at the token claims; API code should rely on the values in AuthData. We use AuthData
// to insulate API code from the differences among Provider tokens.
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

		token, err := GetProvider().VerifyToken(tokenString)
		if err != nil {
			log.Errorf("Unable to verify token; %s", err)
			next.ServeHTTP(w, r)
			return
		}

		var ad AuthData
		if claims, ok := token.Claims.(*CommonClaims); ok && token.Valid {
			// okta token
			switch claims.Issuer {
			case "ssas":
				ad, _ = adFromClaims(claims)
			case "okta":
				var aco, err = GetACOByClientID(claims.ClientID)
				if err != nil {
					log.Errorf("no aco for clientID %s because %v", claims.ClientID, err)
					next.ServeHTTP(w, r)
					return
				}

				db := database.GetGORMDbConnection()
				defer database.Close(db)

				ad.TokenID = claims.Id
				ad.ACOID = aco.UUID.String()
				ad.CMSID = *aco.CMSID
				ad.Blacklisted = aco.Blacklisted

			default:
				var aco, err = GetACOByUUID(claims.ACOID)
				if err != nil {
					log.Errorf("no aco for ACO ID %s because %v", claims.ACOID, err)
					next.ServeHTTP(w, r)
					return
				}
				ad.TokenID = claims.UUID
				ad.ACOID = claims.ACOID
				ad.CMSID = *aco.CMSID
				ad.Blacklisted = aco.Blacklisted
			}
		}
		ctx := context.WithValue(r.Context(), TokenContextKey, token)
		ctx = context.WithValue(ctx, AuthDataContextKey, ad)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value(TokenContextKey)
		if token == nil {
			log.Error("No token found")
			respond(w, http.StatusUnauthorized)
			return
		}

		if token, ok := token.(*jwt.Token); ok {
			err := GetProvider().AuthorizeAccess(token.Raw)
			if err != nil {
				log.Error(err)
				respond(w, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		}
	})
}

// CheckBlacklist checks the auth data is associated with a blacklisted entity
func CheckBlacklist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ad, ok := r.Context().Value(AuthDataContextKey).(AuthData)
		if !ok {
			log.Error()
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Not_found, "AuthData not found")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		if ad.Blacklisted {
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.UnauthorizedErr,
				fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", ad.CMSID))
			responseutils.WriteError(oo, w, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireTokenJobMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ad, ok := r.Context().Value(AuthDataContextKey).(AuthData)
		if !ok {
			log.Error()
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Not_found, "")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		jobID := chi.URLParam(r, "jobID")
		i, err := strconv.Atoi(jobID)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Not_found, "")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		db := database.GetGORMDbConnection()
		defer database.Close(db)

		var job models.Job
		err = db.Find(&job, "id = ? and aco_id = ?", i, ad.ACOID).Error
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.Not_found, "")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respond(w http.ResponseWriter, status int) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, responseutils.TokenErr, "")
	responseutils.WriteError(oo, w, status)
}
