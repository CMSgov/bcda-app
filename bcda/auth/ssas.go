package auth

import (
	"encoding/json"
	"errors"

	"github.com/dgrijalva/jwt-go"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
)

// SSASPlugin is an implementation of Provider that uses the SSAS API.
type SSASPlugin struct {
	client client.SSASClient
}

// RegisterSystem adds a software client for the ACO identified by localID.
func (s SSASPlugin) RegisterSystem(localID string) (Credentials, error) {
	// s.client.CreateSystem()
	return Credentials{}, errors.New("not yet implemented")
}

// UpdateSystem changes data associated with the registered software client identified by clientID.
func (s SSASPlugin) UpdateSystem(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

// DeleteSystem deletes the registered software client identified by clientID, revoking any active tokens.
func (s SSASPlugin) DeleteSystem(clientID string) error {
	return errors.New("Not supported")
}

// ResetSecret creates new or replaces existing credentials for the given clientID.
func (s SSASPlugin) ResetSecret(clientID string) (Credentials, error) {
	// s.client.ResetCredentials()
	return Credentials{}, errors.New("not yet implemented")
}

// RevokeSystemCredentials revokes any existing credentials for the given clientID.
func (s SSASPlugin) RevokeSystemCredentials(ssasID string) error {
	return s.client.DeleteCredentials(ssasID)
}

// MakeAccessToken mints an access token for the given credentials.
func (s SSASPlugin) MakeAccessToken(credentials Credentials) (string, error) {
	ssas, err := client.NewSSASClient()
	if err != nil {
		logger.Errorf("failed to create SSAS client; %s", err.Error())
		return "", err
	}

	ts, err := ssas.GetToken(client.Credentials{ClientID: credentials.ClientID, ClientSecret: credentials.ClientSecret})
	if err != nil {
		logger.Errorf("Failed to get token; %s", err.Error())
		return "", err
	}

	return string(ts), nil
}

// RevokeAccessToken revokes a specific access token identified in a base64-encoded token string.
func (s SSASPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("not yet implemented")
}

// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API.
func (s SSASPlugin) AuthorizeAccess(tokenString string) error {
	return errors.New("not yet implemented")
}

// VerifyToken decodes a base64-encoded token string into a structured token.
func (s SSASPlugin) VerifyToken(tokenString string) (*jwt.Token, error) {
	ssas, err := client.NewSSASClient()
	if err != nil {
		logger.Errorf("failed to create SSAS client; %s", err.Error())
		return nil, err
	}

	b, err := ssas.VerifyPublicToken(tokenString)
	if err != nil {
		logger.Errorf("Failed to verify token; %s", err.Error())
		return nil, err
	}
	var ir map[string]interface{}
	if err = json.Unmarshal(b, &ir); err != nil {
		return nil, err
	}
	if ir["active"] == false {
		return nil, errors.New("inactive token")
	}
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified( tokenString, &jwt.MapClaims{})
	return token, err
}
