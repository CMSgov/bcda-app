package auth

import (
	"github.com/dgrijalva/jwt-go"
)

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	RegisterClient(params []byte) ([]byte, error)
	UpdateClient(params []byte) ([]byte, error)
	DeleteClient(params []byte) error

	GenerateClientCredentials(params []byte) ([]byte, error)
	RevokeClientCredentials(params []byte) error

	RequestAccessToken(params []byte) (jwt.Token, error)
	RevokeAccessToken(tokenString string) error

	ValidateAccessToken(tokenString string) error
	DecodeAccessToken(tokenString string) (jwt.Token, error)
}
