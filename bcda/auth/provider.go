package auth

import (
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
)

const (
	Alpha = "alpha"
	Okta  = "okta"
	SSAS  = "ssas"
)

var providerName = Alpha

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	SetProvider(strings.ToLower(os.Getenv(`BCDA_AUTH_PROVIDER`)))
}

func SetProvider(name string) {
	if name != "" {
		switch strings.ToLower(name) {
		case Okta:
			providerName = name
		case Alpha:
			providerName = name
		case SSAS:
			providerName = name
		default:
			log.Infof(`Unknown providerName %s; using %s`, name, providerName)
		}
	}
	log.Infof(`Auth is made possible by %s`, providerName)
}

func GetProviderName() string {
	return providerName
}

func GetProvider() Provider {
	switch providerName {
	case Alpha:
		return AlphaAuthPlugin{}
	case Okta:
		return NewOktaAuthPlugin(client.NewOktaClient())
	case SSAS:
		c, err := client.NewSSASClient()
		if err != nil {
			log.Fatalf("no client for SSAS; %s", err.Error())
		}
		return SSASPlugin{client: c}
	default:
		return AlphaAuthPlugin{}
	}
}

type AuthData struct {
	ACOID   string
	UserID  string
	TokenID string
}

type Credentials struct {
	UserID       string `json:"user_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Token        Token  `json:"token"`
	ClientName   string `json:"client_name"`
}

// Provider defines operations performed through an authentication provider.
type Provider interface {
	// RegisterSystem adds a software client for the ACO identified by localID.
	RegisterSystem(localID, publicKey string) (Credentials, error)

	// UpdateSystem changes data associated with the registered software client identified by clientID
	UpdateSystem(params []byte) ([]byte, error)

	// DeleteSystem deletes the registered software client identified by clientID, revoking an active tokens
	DeleteSystem(clientID string) error

	// ResetSecret new or replace existing Credentials for the given clientID
	ResetSecret(clientID string) (Credentials, error)

	// RevokeSystemCredentials any existing Credentials for the given clientID
	RevokeSystemCredentials(clientID string) error

	// MakeAccessToken mints an access token for the given credentials
	MakeAccessToken(credentials Credentials) (string, error)

	// RevokeAccessToken a specific access token identified in a base64 encoded token string
	RevokeAccessToken(tokenString string) error

	// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API
	AuthorizeAccess(tokenString string) error

	// VerifyToken decodes a base64 encoded token string into a structured token
	VerifyToken(tokenString string) (*jwt.Token, error)
}
