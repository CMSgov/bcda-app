package auth

import (
	"errors"
)

type AlphaAuthPlugin struct{}

func (p *AlphaAuthPlugin) RegisterClient(params []byte) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) UpdateClient(params []byte) error {
	return errors.New("Not yet implemented")
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

func (p *AlphaAuthPlugin) RequestAccessToken(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) RevokeAccessToken(params []byte) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) ValidateAccessToken(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) DecodeAccessToken(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}
