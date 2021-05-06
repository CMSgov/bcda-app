package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
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
func (s SSASPlugin) MakeAccessToken(credentials Credentials) (string, error) {
	ts, err := s.client.GetToken(client.Credentials{ClientID: credentials.ClientID, ClientSecret: credentials.ClientSecret})
	if err != nil {
		log.SSAS.Errorf("Failed to get token; %s", err.Error())
		return "", err
	}

	return string(ts), nil
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
func adFromClaims(r models.Repository, claims *CommonClaims) (AuthData, error) {
	var (
		ad  AuthData
		err error
	)

	ad.SystemID = claims.SystemID
	ad.ClientID = claims.ClientID
	ad.TokenID = claims.Id

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
	if aco, err = repository.GetACOByCMSID(context.Background(), ad.CMSID); err != nil {
		return ad, fmt.Errorf("no aco for cmsID %s; %v", ad.CMSID, err)
	}
	ad.ACOID = aco.UUID.String()
	ad.Blacklisted = aco.Blacklisted()

	return ad, nil
}

// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API.
func (s SSASPlugin) AuthorizeAccess(tokenString string) error {
	tknEvent := event{op: "AuthorizeAccess"}
	operationStarted(tknEvent)
	t, err := s.VerifyToken(tokenString)
	if err != nil {
		tknEvent.help = fmt.Sprintf("VerifyToken failed in AuthorizeAccess; %s", err.Error())
		operationFailed(tknEvent)
		return err
	}
	claims, ok := t.Claims.(*CommonClaims)
	if !ok {
		return errors.New("invalid ssas claims")
	}
	if _, err = adFromClaims(s.repository, claims); err != nil {
		tknEvent.help = fmt.Sprintf("failed getting AuthData; %s", err.Error())
		operationFailed(tknEvent)
		return err
	}

	operationSucceeded(tknEvent)
	return nil
}

// VerifyToken decodes a base64-encoded token string into a structured token.
func (s SSASPlugin) VerifyToken(tokenString string) (*jwt.Token, error) {
	b, err := s.client.VerifyPublicToken(tokenString)
	if err != nil {
		log.SSAS.Errorf("Failed to verify token; %s", err.Error())
		return nil, err
	}
	var ir map[string]interface{}
	if err = json.Unmarshal(b, &ir); err != nil {
		return nil, err
	}
	if ir["active"] == false {
		return nil, errors.New("inactive or invalid token")
	}
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(tokenString, &CommonClaims{})
	if err != nil {
		return token, err
	}
	claims, ok := token.Claims.(*CommonClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	if claims.Issuer != "ssas" {
		return nil, fmt.Errorf("invalid issuer '%s'", claims.Issuer)
	}
	if claims.Data == "" {
		return nil, errors.New("missing data claim")
	}
	token.Valid = true
	return token, err
}
