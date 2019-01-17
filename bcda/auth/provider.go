package auth

import (
	"github.com/dgrijalva/jwt-go"
)

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	RegisterClient(params ...interface{}) (string, error)
	UpdateClient(params ...interface{}) ([]interface{}, error)
	DeleteClient(params ...interface{}) error

	GenerateClientCredentials(params ...interface{}) (interface{}, error)
	RevokeClientCredentials(params ...interface{}) error

	RequestAccessToken(params ...interface{}) (jwt.Token, error)
	RevokeAccessToken(tokenString string) error

	ValidateAccessToken(tokenString string) error
	DecodeAccessToken(tokenString string) (jwt.Token, error)
}
