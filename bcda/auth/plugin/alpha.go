package auth

import (
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
)

type AlphaAuthPlugin struct{}

func (p *AlphaAuthPlugin) RegisterClient(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) GenerateClientCredentials(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) RevokeClientCredentials(params []byte) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) RequestAccessToken(params []byte) (jwt.Token, error) {
	return jwt.Token{}, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) RevokeAccessToken(token string) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) ValidateAccessToken(token string) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) DecodeAccessToken(token string) (jwt.Token, error) {
	return jwt.Token{}, errors.New("Not yet implemented")
}
