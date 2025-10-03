package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/log"
)

// SSASPlugin is an implementation of Provider that uses the SSAS API.
type SSASPlugin struct {
	client     *client.SSASClient
	repository models.Repository
}

// validates that SSASPlugin implements the interface
var _ Provider = SSASPlugin{}

func (s SSASPlugin) FindAndCreateACOCredentials(ACOID string, ips []string) (string, error) {
	aco, err := s.repository.GetACOByCMSID(context.Background(), ACOID)
	if err != nil {
		return "", err
	}

	// reset client as we need to get updated SSAS_URL
	s.client, err = client.NewSSASClient()
	if err != nil {
		return "", err
	}

	creds, err := s.RegisterSystem(aco.UUID.String(), "", aco.GroupID, ips...)
	if err != nil {
		return "", errors.Wrapf(err, "could not register system for %s", ACOID)
	}

	msg := strings.Join([]string{creds.ClientName, creds.ClientID, creds.ClientSecret, creds.ExpiresAt.String()}, "\n")

	return msg, nil
}

// RegisterSystemWithIPs adds a software client for the ACO identified by localID.
func (s SSASPlugin) RegisterSystem(localID, publicKey, groupID string, ips ...string) (Credentials, error) {
	creds := Credentials{}
	aco, err := s.repository.GetACOByUUID(context.Background(), uuid.Parse(localID))
	if err != nil {
		return creds, errors.Wrap(err, "failed to create system")
	}
	trackingID := uuid.NewRandom().String()

	cb, err := s.client.CreateSystem(
		aco.Name,
		groupID,
		"bcda-api",
		publicKey,
		trackingID,
		ips,
	)
	if err != nil {
		return creds, errors.Wrap(err, "failed to create system")
	}

	err = json.Unmarshal(cb, &creds)
	if err != nil {
		return creds, errors.Wrap(err, "failed to unmarshal response json")
	}

	aco.ClientID = creds.ClientID
	aco.SystemID = creds.SystemID

	if err = s.repository.UpdateACO(context.Background(), aco.UUID,
		map[string]interface{}{"client_id": aco.ClientID, "system_id": aco.SystemID}); err != nil {
		return creds, errors.Wrapf(err, "could not update ACO %s with client and system IDs", *aco.CMSID)
	}

	return creds, nil
}

// GetVersion gets the version of the SSAS client
func (s SSASPlugin) GetVersion() (string, error) {
	return s.client.GetVersion()
}

// ResetSecret creates new or replaces existing credentials for the given ssasID.
func (s SSASPlugin) ResetSecret(clientID string) (Credentials, error) {
	creds := Credentials{}
	aco, err := s.repository.GetACOByClientID(context.Background(), clientID)
	if err != nil {
		return creds, err
	}

	resp, err := s.client.ResetCredentials(aco.SystemID)
	if err != nil {
		return creds, err
	}

	err = json.Unmarshal(resp, &creds)
	if err != nil {
		return creds, err
	}

	return creds, nil
}

// RevokeSystemCredentials revokes any existing credentials for the given clientID.
func (s SSASPlugin) RevokeSystemCredentials(ssasID string) error {
	return s.client.DeleteCredentials(ssasID)
}

// MakeAccessToken mints an access token for the given credentials.
func (s SSASPlugin) MakeAccessToken(credentials Credentials, r *http.Request) (string, error) {
	tokenInfo, err := s.client.GetToken(client.Credentials{ClientID: credentials.ClientID, ClientSecret: credentials.ClientSecret}, *r)
	if err != nil {
		log.SSAS.Errorf("Failed to get token; %s", err.Error())
		return "", err
	}

	return tokenInfo, nil
}

// RevokeAccessToken revokes a specific access token identified in a base64-encoded token string.
func (s SSASPlugin) RevokeAccessToken(tokenString string) error {
	err := s.client.RevokeAccessToken(tokenString)
	if err != nil {
		log.SSAS.Errorf("Failed to revoke token; %s", err.Error())
		return err
	}

	return nil
}

// Extract available values from SSAS token claims.
func (s SSASPlugin) getAuthDataFromClaims(claims *CommonClaims) (AuthData, error) {
	var (
		ad  AuthData
		err error
	)

	ad.SystemID = claims.SystemID
	ad.ClientID = claims.ClientID
	ad.TokenID = claims.ID

	if claims.Data == "" {
		return ad, errors.New("incomplete ssas token")
	}
	d := claims.Data
	if ud, err := strconv.Unquote(claims.Data); err == nil {
		// unquote will fail when the argument string has no quotes to remove!
		d = ud
	}
	type XData struct {
		IDList []string `json:"cms_ids"`
	}
	var xData XData
	if err = json.Unmarshal([]byte(d), &xData); err != nil {
		return ad, fmt.Errorf("can't decode data claim %s; %v", d, err)
	}

	if len(xData.IDList) != 1 {
		return ad, fmt.Errorf("expected one id in list; got %v; source %s", xData.IDList, claims.Data)
	}
	ad.CMSID = xData.IDList[0]

	var aco *models.ACO
	if aco, err = s.repository.GetACOByCMSID(context.Background(), ad.CMSID); err != nil {
		entityNotFoundError := &customErrors.EntityNotFoundError{Err: err, CMSID: ad.CMSID}
		log.SSAS.Errorf(entityNotFoundError.Error())
		return ad, entityNotFoundError
	}
	ad.ACOID = aco.UUID.String()
	ad.Blacklisted = aco.Denylisted()

	return ad, nil
}

// VerifyToken decodes a base64-encoded token string into a structured token,
// verifies token with SSAS and calls check for token expiration.
func (sSASPlugin SSASPlugin) VerifyToken(ctx context.Context, tokenString string) (*jwt.Token, error) {
	token, err := confirmTokenStringLegitimacy(tokenString)
	if err != nil {
		log.SSAS.Errorf("Failed to confirm token string structure/contents; %s", err.Error())
		return token, err
	}

	bytes, err := sSASPlugin.client.CallSSASIntrospect(ctx, tokenString)
	if err != nil {
		log.SSAS.Errorf("Failed to verify token; %s", err.Error())
		return nil, err
	}

	if err := checkTokenExpiration(bytes); err != nil {
		log.SSAS.Errorf("token is inactive; %s", err.Error())
		return nil, err
	}

	token.Valid = true
	return token, nil
}

// confirmTokenStringLegitimacy will take the provided tokenString that was supplied by the requestor
// & ensure it has sound contents/structure before BCDA application performs an API call to SSAS introspect.
func confirmTokenStringLegitimacy(tokenString string) (*jwt.Token, error) {
	token, parseErr := parseTokenStringToJwtToken(tokenString)

	if parseErr != nil {
		return token, parseErr //original logic still returned token (instead of nil), keeping as-is for now
	}
	if err := confirmRequestorTokenPayload(token); err != nil {
		return nil, err
	}
	return token, nil
}

// parseTokenStringToJwtToken will take the provided tokenString that was supplied by the requestor
// and attempt to parse it into a jwt.Token.
func parseTokenStringToJwtToken(tokenString string) (*jwt.Token, error) {
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(tokenString, &CommonClaims{})

	if err != nil {
		return token, &customErrors.RequestorDataError{Err: err, Msg: "unable to parse provided tokenString to jwt.token"}
	}
	return token, nil
}

// confirmRequestorTokenPayload will confirm the jwt.Token contains Claims that adhere to the Common Claims structure
// and that the issuer was from SSAS & Data is not empty.
func confirmRequestorTokenPayload(token *jwt.Token) error {
	tokenPayloadError := errors.New("Incorrect Token Payload From Requestor")
	claims, ok := token.Claims.(*CommonClaims)

	if !ok {
		return &customErrors.RequestorDataError{Err: tokenPayloadError, Msg: "unable to cast token.Claims to CommonClaims struct"}
	} else if claims.Issuer != constants.IssuerSSAS {
		return &customErrors.RequestorDataError{Err: tokenPayloadError, Msg: fmt.Sprintf("invalid issuer supplied in token CommonClaims '%s'", claims.Issuer)}
	} else if claims.Data == constants.EmptyString {
		return &customErrors.RequestorDataError{Err: tokenPayloadError, Msg: "token CommonClaims Data is missing/empty"}
	}
	return nil
}

// checkTokenExpiration parses slice of type byte into map [string] interface & sees if "active" is set to false.
func checkTokenExpiration(bytes []byte) error {
	var introspectResponse map[string]interface{}

	if err := json.Unmarshal(bytes, &introspectResponse); err != nil {
		return &customErrors.InternalParsingError{Err: err, Msg: "unable to unmarshal SSAS introspect response body to json format"}
	}

	if introspectResponse["active"] == false {
		return &customErrors.ExpiredTokenError{Err: errors.New("Expired Token"), Msg: "the provided token has expired (is not active)"}
	}
	return nil
}
