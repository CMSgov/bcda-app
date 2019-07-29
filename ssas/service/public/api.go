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
	"strconv"

	"github.com/go-chi/render"
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

type MFARequest struct {
	CMSID       string  `json:"cms_id"`
	FactorType  string  `json:"factor_type"`
	Passcode    *string `json:"passcode,omitempty"`
	Transaction *string `json:"transaction,omitempty"`
}

type PasswordRequest struct {
	CmsID			string `json:"cms_id"`
	Password 		string `json:"password"`
}

/*
	VerifyPassword is mounted at POST /authn and responds with the account status for a verified username/password
 	combination.
*/
func VerifyPassword(w http.ResponseWriter, r *http.Request) {
	var (
		err				error
		trackingID		string
		passReq			PasswordRequest
	)

	setHeaders(w)

	bodyStr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "invalid_client_metadata", "Request body cannot be read")
		return
	}

	err = json.Unmarshal(bodyStr, &passReq)
	if err != nil {
		service.LogEntrySetField(r,"bodyStr", "<redacted>")
		jsonError(w, "invalid_client_metadata", "Request body cannot be parsed")
		return
	}

	trackingID = uuid.NewRandom().String()
	event := ssas.Event{Op: "VerifyOktaPassword", TrackingID: trackingID, Help: "calling from public.VerifyPassword()"}
	ssas.OperationCalled(event)
	passwordResponse, err := GetProvider().VerifyPassword(passReq.CmsID, passReq.Password, trackingID)
	if err != nil {
		jsonError(w, "invalid_client_metadata", err.Error())
		return
	}

	body, err := json.Marshal(passwordResponse)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		event.Help = "failure generating JSON: " + err.Error()
		ssas.OperationFailed(event)
		return
	}

	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		event.Help = "failure writing response body: " + err.Error()
		ssas.OperationFailed(event)
		return
	}
}

/*
	RequestMultifactorChallenge is mounted at POST /authn/request and sends a multi-factor authentication request
	using the specified factor.

	Valid factor types include:
		"Google TOTP" (Google Authenticator)
		"Okta TOTP"   (Okta Verify app time-based token)
		"Push"        (Okta Verify app push)
		"SMS"
		"Call"
		"Email"

	In the case of the Push factor, a transaction ID is returned to use with the polling endpoint:
	    POST /authn/verify/transactions/{transaction_id}
*/
func RequestMultifactorChallenge(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		trackingID string
		mfaReq     MFARequest
	)

	setHeaders(w)

	bodyStr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "invalid_client_metadata", "Request body cannot be read")
		return
	}

	err = json.Unmarshal(bodyStr, &mfaReq)
	if err != nil {
		service.LogEntrySetField(r, "bodyStr", bodyStr)
		jsonError(w, "invalid_client_metadata", "Request body cannot be parsed")
		return
	}

	trackingID = uuid.NewRandom().String()
	event := ssas.Event{Op: "RequestOktaFactorChallenge", TrackingID: trackingID, Help: "calling from public.RequestMultifactorChallenge()"}
	ssas.OperationCalled(event)
	factorResponse, err := GetProvider().RequestFactorChallenge(mfaReq.CMSID, mfaReq.FactorType, trackingID)
	if err != nil {
		jsonError(w, "invalid_client_metadata", err.Error())
		return
	}

	body, err := json.Marshal(factorResponse)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		event.Help = "failure generating JSON: " + err.Error()
		ssas.OperationFailed(event)
		return
	}

	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		event.Help = "failure writing response body: " + err.Error()
		ssas.OperationFailed(event)
		return
	}
}

/*
	VerifyMultifactorResponse is mounted at POST /authn/verify and tests a multi-factor authentication passcode
	for the specified factor, and should be used for all factor types except Push.
*/
func VerifyMultifactorResponse(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		trackingID string
		mfaReq     MFARequest
		body       []byte
	)

	setHeaders(w)

	bodyStr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "invalid_client_metadata", "Request body cannot be read")
		return
	}

	err = json.Unmarshal(bodyStr, &mfaReq)
	if err != nil {
		service.LogEntrySetField(r, "bodyStr", bodyStr)
		jsonError(w, "invalid_client_metadata", "Request body cannot be parsed")
		return
	}

	if mfaReq.Passcode == nil {
		service.LogEntrySetField(r, "bodyStr", bodyStr)
		jsonError(w, "invalid_client_metadata", "Request body missing passcode")
		return
	}

	trackingID = uuid.NewRandom().String()
	event := ssas.Event{Op: "VerifyOktaFactorResponse", TrackingID: trackingID, Help: "calling from public.VerifyMultifactorResponse()"}
	ssas.OperationCalled(event)
	success := GetProvider().VerifyFactorChallenge(mfaReq.CMSID, mfaReq.FactorType, *mfaReq.Passcode, trackingID)

	if !success {
		event.Help = "passcode rejected"
		ssas.OperationFailed(event)
		body = []byte(`{"factor_result":"failure"}`)
	} else {
		event.Help = "passcode accepted"
		ssas.OperationSucceeded(event)
		body = []byte(`{"factor_result":"success"}`)
	}

	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		event.Help = "failure writing response body: " + err.Error()
		ssas.OperationFailed(event)
		return
	}
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

	setHeaders(w)

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
		service.LogEntrySetField(r, "bodyStr", bodyStr)
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
	event := ssas.Event{Op: "RegisterClient", TrackingID: trackingID, Help: "calling from public.RegisterSystem()"}
	ssas.OperationCalled(event)
	credentials, err := ssas.RegisterSystem(reg.ClientName, rd.GroupID, reg.Scope, publicKeyPEM, trackingID)
	if err != nil {
		jsonError(w, "invalid_client_metadata", err.Error())
		return
	}

	body := []byte(fmt.Sprintf(`{"client_id": "%s","client_secret":"%s","client_secret_expires_at":"%d","client_name":"%s"}`,
		credentials.ClientID, credentials.ClientSecret, credentials.ExpiresAt.Unix(), credentials.ClientName))
	// https://tools.ietf.org/html/rfc7591#section-3.2 dictates 201, not 200
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		event.Help = "failure writing response body: " + err.Error()
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

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   string `json:"expires_in"`
}

func token(w http.ResponseWriter, r *http.Request) {
	clientID, secret, ok := r.BasicAuth()
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	system, err := ssas.GetSystemByClientID(clientID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	savedSecret, err := system.GetSecret()
	if err != nil || !ssas.Hash(savedSecret).IsHashOf(secret) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	token, ts, err := server.MintToken(system.GroupID, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// https://tools.ietf.org/html/rfc6749#section-5.1
	// expires_in is duration in seconds
	expiresIn := token.Claims.(service.CommonClaims).ExpiresAt - token.Claims.(service.CommonClaims).IssuedAt
	m := TokenResponse{AccessToken: ts, TokenType: "bearer", ExpiresIn: strconv.FormatInt(expiresIn, 10)}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	render.JSON(w, r, m)
}
