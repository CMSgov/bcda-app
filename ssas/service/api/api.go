package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/auth/rsautils"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/pborman/uuid"
	"io/ioutil"
	"net/http"
)

type Key struct {
	E string `json:"e"`
	N string `json:"n"`
	KTY string `json:"kty"`
	Use string `json:"use,omitempty"`
}

type JWKS struct {
	Keys []Key `json:"keys"`
}

type RegistrationRequest struct {
	ClientID string `json:"client_id"`
	ClientName string `json:"client_name"`
	Scope string `json:"scope,omitempty"`
	JSONWebKeys JWKS `json:"jwks"`
}

// POST /auth/register
func RegisterSystem(w http.ResponseWriter, r *http.Request) {
	var (
		rd  auth.AuthRegData
		err error
		reg RegistrationRequest
		publicKeyBytes []byte
		trackingID string
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if rd, err = readRegData(r); err != nil {
		// Specified in RFC 7592 https://tools.ietf.org/html/rfc7592#page-6
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	bodyStr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// Response types and format specified in RFC 7591 https://tools.ietf.org/html/rfc7591#section-3.2.2
		jsonError(w, "invalid_client_metadata", "Request body cannot be read")
		return
	}

	err = json.Unmarshal(bodyStr, &reg)
	if err != nil {
		jsonError(w, "invalid_client_metadata", "Request body cannot be parsed")
		return
	}

	if reg.JSONWebKeys.Keys == nil || len(reg.JSONWebKeys.Keys) > 1 {
		jsonError(w, "invalid_client_metadata", "Exactly one JWK must be presented")
		return
	}

	publicKeyBytes, err = json.Marshal(reg.JSONWebKeys.Keys[0])
	if err != nil {
		jsonError(w, "invalid_client_metadata", "Unable to read JWK")
		return
	}

	publicKeyPEM, err := rsautils.ConvertJWKToPEM(string(publicKeyBytes))
	if err != nil {
	//	jsonError(w, "invalid_client_metadata", "Unable to process JWK")
		jsonError(w, "invalid_client_metadata", err.Error())
		return
	}

	// Log the source of the call for this operation.  Remaining logging will be in ssas.RegisterSystem() below.
	trackingID = uuid.NewRandom().String()
	ssas.OperationCalled(ssas.Event{Op: "RegisterClient", TrackingID: trackingID})
	credentials, err := ssas.RegisterSystem(reg.ClientName, rd.GroupID, reg.Scope, publicKeyPEM, trackingID)
	if err != nil {
		jsonError(w, "invalid_client_metadata", err.Error())
		return
	}

	body := []byte(fmt.Sprintf(`{"client_id": "%s","client_secret":"%s","client_secret_expires_at":"%d","client_name":"%s"}`,
		credentials.ClientID, credentials.ClientSecret, credentials.ExpiresAt.Unix(), credentials.ClientName))
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func readRegData(r *http.Request) (data auth.AuthRegData, err error) {
	var ok bool
	data, ok = r.Context().Value("rd").(auth.AuthRegData)
	if !ok {
		err = errors.New("no registration data in context")
	}
	return
}

func jsonError(w http.ResponseWriter, error string, description string) {
	w.WriteHeader(http.StatusBadRequest)
	body := []byte(fmt.Sprintf(`{"error":"%s","error_description":"%s"}`,error, description))
	_, err := w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}