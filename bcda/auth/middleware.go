package auth

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/database"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	responseutilsv2 "github.com/CMSgov/bcda-app/bcda/responseutils/v2"
	"github.com/CMSgov/bcda-app/log"
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

		// ParseToken is called on every request, but not every request has a token
		// Continue serving if not Auth token is found and let RequireToken throw the error
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		rw := getRespWriter(r.URL.Path)

		authRegexp := regexp.MustCompile(`^Bearer (\S+)$`)
		authSubmatches := authRegexp.FindStringSubmatch(authHeader)
		if len(authSubmatches) < 2 {
			log.Auth.Warn("Invalid Authorization header value")
			rw.Exception(w, http.StatusUnauthorized, responseutils.TokenErr, "")
			return
		}

		tokenString := authSubmatches[1]

		token, ad, err := AuthorizeAccess(tokenString)
		if err != nil {
			handleTokenVerificationError(w, rw, err)
			return
		}

		ctx := context.WithValue(r.Context(), TokenContextKey, token)
		ctx = context.WithValue(ctx, AuthDataContextKey, ad)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API.
func AuthorizeAccess(tokenString string) (*jwt.Token, AuthData, error) {
	tknEvent := event{op: "AuthorizeAccess"}
	operationStarted(tknEvent)
	token, err := GetProvider().VerifyToken(tokenString)

	var ad AuthData

	if err != nil {
		tknEvent.help = fmt.Sprintf("VerifyToken failed in AuthorizeAccess; %s", err.Error())
		operationFailed(tknEvent)
		return nil, ad, err
	}

	claims, ok := token.Claims.(*CommonClaims)
	if !ok || !token.Valid {
		// These should already trigger an error within VerifyToken, so in theory it's unreachable code.
		return nil, ad, errors.New("invalid ssas claims")
	}

	ad, err = GetProvider().getAuthDataFromClaims(claims)
	if err != nil {
		tknEvent.help = fmt.Sprintf("failed getting AuthData; %s", err.Error())
		operationFailed(tknEvent)
		return nil, ad, err
	}

	operationSucceeded(tknEvent)
	return token, ad, nil
}

func handleTokenVerificationError(w http.ResponseWriter, rw fhirResponseWriter, err error) {
	if err != nil {
		log.Auth.Error(err)

		switch err.(type) {
		case *customErrors.ExpiredTokenError:
			rw.Exception(w, http.StatusUnauthorized, responseutils.ExpiredErr, "")
		case *customErrors.EntityNotFoundError:
			rw.Exception(w, http.StatusForbidden, responseutils.UnauthorizedErr, responseutils.UnknownEntityErr)
		case *customErrors.RequestorDataError:
			rw.Exception(w, http.StatusBadRequest, responseutils.InternalErr, "")
		case *customErrors.RequestTimeoutError:
			rw.Exception(w, http.StatusServiceUnavailable, responseutils.InternalErr, "")
		case *customErrors.ConfigError, *customErrors.InternalParsingError, *customErrors.UnexpectedSSASError:
			rw.Exception(w, http.StatusInternalServerError, responseutils.InternalErr, "")
		default:
			rw.Exception(w, http.StatusUnauthorized, responseutils.TokenErr, "")
		}
	}
}

// Verify that a token was verified and stored in the request context.
// This depends on ParseToken being called beforehand in the routing middleware.
func RequireTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := getRespWriter(r.URL.Path)

		token := r.Context().Value(TokenContextKey)
		if token == nil {
			log.Auth.Error("No token found")
			rw.Exception(w, http.StatusUnauthorized, responseutils.TokenErr, "")
			return
		}

		if _, ok := token.(*jwt.Token); ok {
			next.ServeHTTP(w, r)
		}
	})
}

// CheckBlacklist checks the auth data is associated with a blacklisted entity
func CheckBlacklist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := getRespWriter(r.URL.Path)

		ad, ok := r.Context().Value(AuthDataContextKey).(AuthData)
		if !ok {
			log.Auth.Error()
			rw.Exception(w, http.StatusNotFound, responseutils.NotFoundErr, "AuthData not found")
			return
		}

		if ad.Blacklisted {
			rw.Exception(w, http.StatusForbidden, responseutils.UnauthorizedErr, fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", ad.CMSID))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireTokenJobMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := getRespWriter(r.URL.Path)

		ad, ok := r.Context().Value(AuthDataContextKey).(AuthData)
		if !ok {
			log.Auth.Error()
			rw.Exception(w, http.StatusNotFound, responseutils.NotFoundErr, "AuthData not found")
			return
		}

		jobID, err := strconv.ParseUint(chi.URLParam(r, "jobID"), 10, 64)
		if err != nil {
			log.Auth.Error(err)
			rw.Exception(w, http.StatusNotFound, responseutils.NotFoundErr, err.Error())
			return
		}

		repository, newRelicCtx := postgres.NewRepositoryWithContext(database.Connection, r.Context())

		job, err := repository.GetJobByID(newRelicCtx, uint(jobID))
		if err != nil {
			log.Auth.Error(err)
			rw.Exception(w, http.StatusNotFound, responseutils.NotFoundErr, "")
			return
		}

		// ACO did not create the job
		if !strings.EqualFold(ad.ACOID, job.ACOID.String()) {
			log.Auth.Errorf("ACO %s does not have access to job ID %d %s",
				ad.ACOID, job.ID, job.ACOID)
			rw.Exception(w, http.StatusNotFound, responseutils.NotFoundErr, "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type fhirResponseWriter interface {
	Exception(http.ResponseWriter, int, string, string)
	NotFound(http.ResponseWriter, int, string, string)
}

func getRespWriter(path string) fhirResponseWriter {
	if strings.Contains(path, "/v1/") {
		return responseutils.NewResponseWriter()
	} else if strings.Contains(path, "/v2/") {
		return responseutilsv2.NewResponseWriter()
	} else {
		// CommonAuth is used in requests not exclusive to v1 or v2 (ie data requests or /_version).
		// In the cases we cannot discern a version we default to v1
		return responseutils.NewResponseWriter()
	}
}
