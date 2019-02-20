package auth

import (
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
)

type OktaAuthPlugin struct{}

func (o OktaAuthPlugin) RegisterClient(localID string) (Credentials, error) {
	if localID == "" {
		return Credentials{}, errors.New("you must provide a localID")
	}

	// using the mocktaclient here for now
	id, key, err := addClientApplication(localID)
	return Credentials{
		ClientID:     id,
		ClientSecret: key,
	}, err
}

func (o OktaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (o OktaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) GenerateClientCredentials(clientID string) (Credentials, error) {
	return Credentials{}, errors.New("not yet implemented")
}

func (o OktaAuthPlugin) RevokeClientCredentials(params []byte) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) RequestAccessToken(params []byte) (Token, error) {
	return Token{}, errors.New("not yet implemented")
}

func (o OktaAuthPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) ValidateJWT(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) DecodeJWT(tokenString string) (jwt.Token, error) {
	return jwt.Token{}, errors.New("not yet implemented")
}
