package auth

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
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

	token, err := GetProvider().MakeAccessToken(Credentials{ClientID: clientId, ClientSecret: secret})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1
	// not included: recommended field expires_in
	body := []byte(fmt.Sprintf(`{"access_token": "%s","token_type":"bearer"}`, token))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	log.WithField("client_id", clientId).Println("issued access token")
}

func GetAuthGroups(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func CreateAuthGroup(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func EditAuthGroup(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func DeactivateAuthGroup(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}
