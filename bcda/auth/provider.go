package auth

import "github.com/dgrijalva/jwt-go"

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	RegisterClient(params []byte) ([]byte, error)
	UpdateClient(params []byte) ([]byte, error)
	DeleteClient(params []byte) error

	GenerateClientCredentials(params []byte) ([]byte, error)
	RevokeClientCredentials(params []byte) error

	RequestAccessToken(params []byte) (Token, error)
	RevokeAccessToken(tokenString string) error

	ValidateJwtToken(tokenString string) error
	DecodeJwtToken(tokenString string) (jwt.Token, error)
}
