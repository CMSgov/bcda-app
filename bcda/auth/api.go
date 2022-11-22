package auth

import (
	"net/http"
	"strconv"

	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in,omitempty"`
	TokenType   string `json:"token_type"`
}

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
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case *customErrors.UnexpectedSSASError:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		default:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		return
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_, err = w.Write([]byte(tokenInfo))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
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
