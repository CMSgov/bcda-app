package auth

import (
	"net/http"

	"strconv"

	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/middleware"
	"github.com/sirupsen/logrus"

	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
)

/*
	swagger:route POST /auth/token auth GetAuthToken

	Get access token

	Verifies Basic authentication credentials, and returns a JWT bearer token that can be presented to the other API endpoints.

	Produces:
	- application/json

	Schemes: https

	Security:
		basic_auth:

	Responses:
		200: tokenResponse
		400: missingCredentials
		401: invalidCredentials
		500: serverError
*/

type BaseApi struct {
	provider Provider
}

func NewBaseApi(provider Provider) BaseApi {
	return BaseApi{provider: provider}
}

func (a BaseApi) GetAuthToken(w http.ResponseWriter, r *http.Request) {
	ctxLogger := log.API.WithFields(logrus.Fields{"transaction_id": r.Context().Value(middleware.CtxTransactionKey)})

	clientId, secret, ok := r.BasicAuth()
	if !ok {
		ctxLogger.WithField("resp_status", http.StatusBadRequest).Errorf("Error Basic Authentication - HTTPS Status Code: %v", http.StatusBadRequest)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	tokenInfo, err := a.provider.MakeAccessToken(Credentials{ClientID: clientId, ClientSecret: secret}, r)
	if err != nil {
		switch err.(type) {
		case *customErrors.RequestTimeoutError:
			//default retrySeconds: 1 second (may convert to environmental variable later)
			retrySeconds := strconv.FormatInt(int64(1), 10)
			w.Header().Set("Retry-After", retrySeconds)
			ctxLogger.WithField("resp_status", http.StatusServiceUnavailable).Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusServiceUnavailable)
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		case *customErrors.InternalParsingError:
			ctxLogger.WithField("resp_status", http.StatusInternalServerError).Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusInternalServerError)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		case *customErrors.SSASErrorUnauthorized:
			ctxLogger.WithField("resp_status", http.StatusUnauthorized).Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusUnauthorized)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		case *customErrors.SSASErrorBadRequest:
			ctxLogger.WithField("resp_status", http.StatusBadRequest).Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusBadRequest)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		default:
			ctxLogger.WithField("resp_status", http.StatusInternalServerError).Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusInternalServerError)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_, err = w.Write([]byte(tokenInfo))
	if err != nil {
		ctxLogger.WithField("resp_status", http.StatusInternalServerError).Errorf("Error writing response - %s | HTTPS Status Code: %v", err.Error(), http.StatusInternalServerError)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

/*
swagger:route GET /auth/welcome auth welcome

# Test authentication

If a valid token is presented, show a welcome message.

Produces:
- application/json

Schemes: http, https

Security:

	bearer_token:

Responses:

	200: welcome
	401: invalidCredentials
*/
func (a BaseApi) Welcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success":"Welcome to the Beneficiary Claims Data API!"}`))
}
