package auth

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
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

		// TODO (BCDA-3412): Remove this reference once we've captured all of the necessary
		// logic into a service method.
		db := database.Connection

		repository := postgres.NewRepository(db)

		var ad AuthData
		if claims, ok := token.Claims.(*CommonClaims); ok && token.Valid {
			switch claims.Issuer {
			case "ssas":
				ad, _ = adFromClaims(repository, claims)
			default:
				log.Errorf("Unsupported claims issuer %s", claims.Issuer)
				respond(w, http.StatusNotFound)
				return
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
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, "AuthData not found")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		if ad.Blacklisted {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.UnauthorizedErr, fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", ad.CMSID))
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
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, "AuthData not found")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		jobID, err := strconv.ParseUint(chi.URLParam(r, "jobID"), 10, 64)
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, err.Error())
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		repository := postgres.NewRepository(database.Connection)

		job, err := repository.GetJobByID(context.Background(), uint(jobID))
		if err != nil {
			log.Error(err)
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, "")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}

		// ACO did not create the job
		if !strings.EqualFold(ad.ACOID, job.ACOID.String()) {
			log.Errorf("ACO %s does not have access to job ID %d %s",
				ad.ACOID, job.ID, job.ACOID)
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.NotFoundErr, "")
			responseutils.WriteError(oo, w, http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respond(w http.ResponseWriter, status int) {
	oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.TokenErr, "")
	responseutils.WriteError(oo, w, status)
}
