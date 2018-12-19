package auth

import (
	jwt "github.com/dgrijalva/jwt-go"
)

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	RegisterClient(params []byte) ([]byte, error)
	UpdateClient(params []byte) ([]byte, error)
	DeleteClient(params []byte) error

	GenerateClientCredentials(params []byte) ([]byte, error)
	RevokeClientCredentials(params []byte) error

	RequestAccessToken(params []byte) (jwt.Token, error)
	RevokeAccessToken(token string) error

	ValidateAccessToken(token string) error
	DecodeAccessToken(token string) (jwt.Token, error)
}
