package auth

import (
	"os"

	jwt "github.com/dgrijalva/jwt-go"
)

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	// Ask the auth Provider to register a software client for the ACO identified by localID.
	RegisterClient(params []byte) ([]byte, error)

	// Update data associated with the registered software client identified by clientID
	UpdateClient(params []byte) ([]byte, error)

	// Delete the registered software client identified by clientID, revoking an active tokens
	DeleteClient(params []byte) error

	// Generate new or replace existing Credentials for the given clientID
	GenerateClientCredentials(params []byte) ([]byte, error)

	// Revoke any existing Credentials for the given clientID
	RevokeClientCredentials(params []byte) error

	// Request an access token with a specific time-to-live for the given clientID
	RequestAccessToken(params []byte) (Token, error)

	// Revoke a specific access token identified in a base64 encoded token string
	RevokeAccessToken(tokenString string) error

	// Assert that a base64 encoded token string is valid for accessing the BCDA API
	ValidateJWT(tokenString string) error

	// Decode a base64 encoded token string
	DecodeJWT(tokenString string) (jwt.Token, error)
}

func GetProvider() Provider {
	v := os.Getenv("BCDA_AUTH_PROVIDER")
	switch v {
	case "Alpha":
		return new(AlphaAuthPlugin)
	case "Okta":
		return new(OktaAuthPlugin)
	default:
		return new(AlphaAuthPlugin)
	}
}
