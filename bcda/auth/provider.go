package auth

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	RegisterClient(params []byte) ([]byte, error)
	UpdateClient(params []byte) ([]byte, error)
	DeleteClient(params []byte) error

	GenerateClientCredentials(params []byte) ([]byte, error)
	RevokeClientCredentials(params []byte) error

	RequestAccessToken(params []byte) ([]byte, error)
	RevokeAccessToken(params []byte) error

	ValidateAccessToken(params []byte) ([]byte, error)
	DecodeAccessToken(params []byte) ([]byte, error)
}
