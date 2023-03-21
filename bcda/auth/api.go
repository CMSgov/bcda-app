package auth

import (
	"net/http"

	"strconv"

	"github.com/CMSgov/bcda-app/log"

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

func GetAuthToken(w http.ResponseWriter, r *http.Request) {
	clientId, secret, ok := r.BasicAuth()
	if !ok {
		log.API.Errorf("Error Basic Authentication - HTTPS Status Code: %v", http.StatusBadRequest)

		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	tokenInfo, err := GetProvider().MakeAccessToken(Credentials{ClientID: clientId, ClientSecret: secret})
	if err != nil {

		switch err.(type) {
		case *customErrors.RequestTimeoutError:
			//default retrySeconds: 1 second (may convert to environmental variable later)
			retrySeconds := strconv.FormatInt(int64(1), 10)
			w.Header().Set("Retry-After", retrySeconds)

			log.API.Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusServiceUnavailable)

			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		case *customErrors.UnexpectedSSASError, *customErrors.InternalParsingError:
			log.API.Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusInternalServerError)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		case *customErrors.SSASErrorUnauthorized:
			log.API.Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusUnauthorized)

			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		case *customErrors.SSASErrorBadRequest:
			log.API.Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusBadRequest)

			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		default:
			log.API.Errorf("Error making access token - %s | HTTPS Status Code: %v", err.Error(), http.StatusUnauthorized)

			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
		return
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_, err = w.Write([]byte(tokenInfo))
	if err != nil {
		log.API.Errorf("Error writing response - %s | HTTPS Status Code: %v", err.Error(), http.StatusInternalServerError)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

/*
	swagger:route GET /auth/welcome auth welcome

	Test authentication

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
func Welcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success":"Welcome to the Beneficiary Claims Data API!"}`))
}
