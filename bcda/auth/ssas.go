package auth

import (
	"errors"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	jwt "github.com/dgrijalva/jwt-go"
)

// SSASPlugin is an implementation of Provider that uses the SSAS API.
type SSASPlugin struct {
	client client.SSASClient
}

// RegisterSystem adds a software client for the ACO identified by localID.
func (s SSASPlugin) RegisterSystem(localID string) (Credentials, error) {
	// s.client.CreateSystem()
	return Credentials{}, errors.New("Not yet implemented")
}

// UpdateSystem changes data associated with the registered software client identified by clientID.
func (s SSASPlugin) UpdateSystem(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

// DeleteSystem deletes the registered software client identified by clientID, revoking any active tokens.
func (s SSASPlugin) DeleteSystem(clientID string) error {
	return errors.New("Not yet implemented")
}

// ResetSecret creates new or replaces existing credentials for the given ssasID.
func (s SSASPlugin) ResetSecret(ssasID string) (Credentials, error) {
	// s.client.ResetCredentials()
	return Credentials{}, errors.New("Not yet implemented")
}

// RevokeSystemCredentials revokes any existing credentials for the given ssasID.
func (s SSASPlugin) RevokeSystemCredentials(clientID string) error {
	// s.client.DeleteCredentials()
	return errors.New("Not yet implemented")
}

// MakeAccessToken mints an access token for the given credentials.
func (s SSASPlugin) MakeAccessToken(credentials Credentials) (string, error) {
	return "", errors.New("Not yet implemented")
}

// RevokeAccessToken revokes a specific access token identified in a base64-encoded token string.
func (s SSASPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("Not yet implemented")
}

// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API.
func (s SSASPlugin) AuthorizeAccess(tokenString string) error {
	return errors.New("Not yet implemented")
}

// VerifyToken decodes a base64-encoded token string into a structured token.
func (s SSASPlugin) VerifyToken(tokenString string) (*jwt.Token, error) {
	return nil, errors.New("Not yet implemented")
}
