package auth

import (
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
)

type OktaAuthPlugin struct{}

func (o *OktaAuthPlugin) RegisterClient(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) GenerateClientCredentials(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) RevokeClientCredentials(params []byte) error {
	return errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) RequestAccessToken(params []byte) (Token, error) {
	return Token{}, errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) ValidateJWT(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o *OktaAuthPlugin) DecodeJWT(tokenString string) (jwt.Token, error) {
	return jwt.Token{}, errors.New("not yet implemented")
}
