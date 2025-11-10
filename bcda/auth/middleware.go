package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models"
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

type AuthMiddleware struct {
	provider Provider
}

func NewAuthMiddleware(provider Provider) AuthMiddleware {
	return AuthMiddleware{provider: provider}
}

// ParseToken puts the decoded token and AuthData value into the request context. Decoded values come from
// tokens verified by our provider as correct and unexpired. Tokens may be presented in requests to
// unauthenticated endpoints (mostly swagger?). We still want to extract the token data for logging purposes,
// even when we don't use it for authorization. Authorization for protected endpoints occurs in RequireTokenAuth().
// Only auth code should look at the token claims; API code should rely on the values in AuthData. We use AuthData
// to insulate API code from the differences among Provider tokens.
func (m AuthMiddleware) ParseToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
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
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: Invalid Authorization header value", responseutils.TokenErr),
				logrus.Fields{"resp_status": http.StatusUnauthorized},
			)
			rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusUnauthorized, responseutils.TokenErr, responseutils.TokenErr)
			return
		}

		tokenString := authSubmatches[1]

		token, ad, err := m.AuthorizeAccess(ctx, tokenString)
		if err != nil {
			handleTokenVerificationError(log.NewStructuredLoggerEntry(log.Auth, ctx), w, rw, err)
			return
		}

		ctx = context.WithValue(ctx, TokenContextKey, token)
		ctx = context.WithValue(ctx, AuthDataContextKey, ad)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API.
func (m AuthMiddleware) AuthorizeAccess(ctx context.Context, tokenString string) (*jwt.Token, AuthData, error) {
	tknEvent := event{op: "AuthorizeAccess"}
	operationStarted(tknEvent)
	token, err := m.provider.VerifyToken(ctx, tokenString)

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

	ad, err = m.provider.getAuthDataFromClaims(claims)
	if err != nil {
		tknEvent.help = fmt.Sprintf("failed getting AuthData; %s", err.Error())
		operationFailed(tknEvent)
		return nil, ad, err
	}

	operationSucceeded(tknEvent)
	return token, ad, nil
}

func handleTokenVerificationError(ctx context.Context, w http.ResponseWriter, rw fhirResponseWriter, err error) {
	if err != nil {
		switch err.(type) {
		case *customErrors.ExpiredTokenError:
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Verification error: %+v", responseutils.ExpiredErr, err),
				logrus.Fields{"resp_status": http.StatusUnauthorized},
			)
			rw.Exception(ctx, w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), responseutils.ExpiredErr)
		case *customErrors.EntityNotFoundError:
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Verification error: %+v", responseutils.UnauthorizedErr, err),
				logrus.Fields{"resp_status": http.StatusForbidden},
			)
			rw.Exception(ctx, w, http.StatusForbidden, http.StatusText(http.StatusForbidden), responseutils.UnauthorizedErr)
		case *customErrors.RequestorDataError:
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Verification error: %+v", responseutils.RequestErr, err),
				logrus.Fields{"resp_status": http.StatusBadRequest},
			)
			rw.Exception(ctx, w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest), responseutils.RequestErr)
		case *customErrors.RequestTimeoutError:
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: Verification error: %+v", responseutils.InternalErr, err),
				logrus.Fields{"resp_status": http.StatusServiceUnavailable},
			)
			rw.Exception(ctx, w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable), responseutils.InternalErr)
		case *customErrors.ConfigError, *customErrors.InternalParsingError, *customErrors.UnexpectedSSASError:
			ctx, _ = log.ErrorExtra(
				ctx,
				fmt.Sprintf("%s: Verification error: %+v", responseutils.InternalErr, err),
				logrus.Fields{"resp_status": http.StatusInternalServerError},
			)
			rw.Exception(ctx, w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), responseutils.InternalErr)
		default:
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: Verification error: %+v", responseutils.TokenErr, err),
				logrus.Fields{"resp_status": http.StatusUnauthorized},
			)
			rw.Exception(ctx, w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), responseutils.TokenErr)
		}
	}
}

// Verify that a token was verified and stored in the request context.
// This depends on ParseToken being called beforehand in the routing middleware.
func RequireTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := getRespWriter(r.URL.Path)
		ctx := r.Context()

		token := ctx.Value(TokenContextKey)
		if token == nil {
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: No token found", responseutils.TokenErr),
				logrus.Fields{"resp_status": http.StatusUnauthorized},
			)
			rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), responseutils.TokenErr)
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
		ctx := r.Context()

		ad, ok := ctx.Value(AuthDataContextKey).(AuthData)
		if !ok {
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: AuthData not found", responseutils.NotFoundErr),
				logrus.Fields{"resp_status": http.StatusNotFound},
			)
			rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusNotFound, responseutils.NotFoundErr, "AuthData not found")
			return
		}

		if ad.Blacklisted {
			ctx, _ = log.WarnExtra(
				ctx,
				fmt.Sprintf("%s: ACO %s is denylisted: ", responseutils.UnauthorizedErr, ad.CMSID),
				logrus.Fields{"resp_status": http.StatusForbidden},
			)
			rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusForbidden, responseutils.UnauthorizedErr, fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", ad.CMSID))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m AuthMiddleware) RequireTokenJobMatch(db *sql.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rw := getRespWriter(r.URL.Path)
			ctx := r.Context()

			ad, ok := ctx.Value(AuthDataContextKey).(AuthData)
			if !ok {
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: AuthData not found", responseutils.UnauthorizedErr),
					logrus.Fields{"resp_status": http.StatusUnauthorized},
				)
				rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusUnauthorized, responseutils.UnauthorizedErr, "AuthData not found")
				return
			}

			//Throw an invalid request for non-unsigned integers
			jobID, err := strconv.ParseUint(chi.URLParam(r, "jobID"), 10, 64)
			if err != nil {
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: Failed to parse jobID: %+v", responseutils.RequestErr, err),
					logrus.Fields{"resp_status": http.StatusBadRequest},
				)
				rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusBadRequest, responseutils.RequestErr, "")
				return
			}

			repository := postgres.NewRepository(db)

			job, err := repository.GetJobByID(ctx, uint(jobID))
			if err != nil {
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: Job not found, ID: %+v", responseutils.NotFoundErr, jobID),
					logrus.Fields{"resp_status": http.StatusNotFound},
				)
				rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusNotFound, responseutils.NotFoundErr, "")
				return
			}

			if job.Status == models.JobStatusExpired || job.Status == models.JobStatusArchived {
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: Job found but expired or archived, ID: %+v", responseutils.JobExpiredErr, jobID),
					logrus.Fields{"resp_status": http.StatusNotFound},
				)
				rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusNotFound, responseutils.JobExpiredErr, "")
			}

			// ACO did not create the job
			if !strings.EqualFold(ad.ACOID, job.ACOID.String()) {
				log.Auth.Errorf("ACO %s does not have access to job ID %d %s",
					ad.ACOID, job.ID, job.ACOID)
				ctx, _ = log.WarnExtra(
					ctx,
					fmt.Sprintf("%s: ACO %s does not have access to job ID %d (ACO ID of job: %s)", responseutils.UnauthorizedErr, ad.ACOID, jobID, job.ACOID),
					logrus.Fields{"resp_status": http.StatusUnauthorized},
				)
				rw.Exception(log.NewStructuredLoggerEntry(log.Auth, ctx), w, http.StatusUnauthorized, responseutils.UnauthorizedErr, "")
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

type fhirResponseWriter interface {
	Exception(context.Context, http.ResponseWriter, int, string, string)
	NotFound(context.Context, http.ResponseWriter, int, string, string)
}

func getRespWriter(path string) fhirResponseWriter {
	if strings.Contains(path, "/v1/") {
		return responseutils.NewFhirResponseWriter()
	} else if strings.Contains(path, "/v2/") {
		return responseutilsv2.NewFhirResponseWriter()
	} else if strings.Contains(path, fmt.Sprintf("/%s/", constants.V3Version)) {
		return responseutilsv2.NewFhirResponseWriter() // TODO: V3
	} else {
		// CommonAuth is used in requests not exclusive to v1 or v2 (ie data requests or /_version).
		// In the cases we cannot discern a version we default to v1
		return responseutils.NewFhirResponseWriter()
	}
}
