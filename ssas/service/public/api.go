/*
	Package public (ssas/service/api/public) contains API functions, middleware, and a router designed to:
		1. Be accessible to the public
		2. Offer system self-registration and self-management
*/
package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
)

type Key struct {
	E   string `json:"e"`
	N   string `json:"n"`
	KTY string `json:"kty"`
	Use string `json:"use,omitempty"`
}

type JWKS struct {
	Keys []Key `json:"keys"`
}

type RegistrationRequest struct {
	ClientID    string `json:"client_id"`
	ClientName  string `json:"client_name"`
	Scope       string `json:"scope,omitempty"`
	JSONWebKeys JWKS   `json:"jwks"`
}

/*
	RegisterSystem is mounted at POST /auth/register and allows for self-registration.  It requires that a
	registration token containing one or more group ids be presented and parsed by middleware, with the
    GroupID[s] placed in the context key "rd".
*/
func RegisterSystem(w http.ResponseWriter, r *http.Request) {
	var (
		rd             ssas.AuthRegData
		err            error
		reg            RegistrationRequest
		publicKeyBytes []byte
		trackingID     string
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if rd, err = readRegData(r); err != nil || rd.GroupID == "" {
		service.GetLogEntry(r).Println("missing or invalid GroupID")
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
		service.LogEntrySetField(r,"bodyStr", bodyStr)
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

	publicKeyPEM, err := ssas.ConvertJWKToPEM(string(publicKeyBytes))
	if err != nil {
		jsonError(w, "invalid_client_metadata", "Unable to process JWK")
		return
	}

	// Log the source of the call for this operation.  Remaining logging will be in ssas.RegisterSystem() below.
	trackingID = uuid.NewRandom().String()
	event := ssas.Event{Op: "RegisterClient", TrackingID: trackingID}
	ssas.OperationCalled(event)
	credentials, err := ssas.RegisterSystem(reg.ClientName, rd.GroupID, reg.Scope, publicKeyPEM, trackingID)
	if err != nil {
		jsonError(w, "invalid_client_metadata", err.Error())
		return
	}

	body := []byte(fmt.Sprintf(`{"client_id": "%s","client_secret":"%s","client_secret_expires_at":"%d","client_name":"%s"}`,
		credentials.ClientID, credentials.ClientSecret, credentials.ExpiresAt.Unix(), credentials.ClientName))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		ssas.OperationFailed(event)
		return
	}
}

func readRegData(r *http.Request) (data ssas.AuthRegData, err error) {
	var ok bool
	data, ok = r.Context().Value("rd").(ssas.AuthRegData)
	if !ok {
		err = errors.New("no registration data in context")
	}
	return
}

func jsonError(w http.ResponseWriter, error string, description string) {
	w.WriteHeader(http.StatusBadRequest)
	body := []byte(fmt.Sprintf(`{"error":"%s","error_description":"%s"}`, error, description))
	_, err := w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
