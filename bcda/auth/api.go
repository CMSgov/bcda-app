package auth

import (
	"fmt"
	"net/http"

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
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	token, expiresIn, err := GetProvider().MakeAccessToken(Credentials{ClientID: clientId, ClientSecret: secret})
	if err != nil {
		switch err.(type) {
		case *customErrors.RequestTimeoutError:
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case *customErrors.UnexpectedSSASError:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		default:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1

	// move to a helper function
	body := []byte(fmt.Sprintf(`{"access_token": "%s", "expires_in": "%s", "token_type":"bearer"}`, token, expiresIn))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_, err = w.Write(body)
	if err != nil {
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
