package auth

import (
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
)

const (
	Alpha = "alpha"
	Okta  = "okta"
	SSAS  = "ssas"
)

var providerName = Alpha
var repository models.Repository

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	SetProvider(strings.ToLower(os.Getenv(`BCDA_AUTH_PROVIDER`)))

	repository = postgres.NewRepository(database.GetDbConnection())
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
		return AlphaAuthPlugin{repository}
	case Okta:
		return NewOktaAuthPlugin(client.NewOktaClient())
	case SSAS:
		c, err := client.NewSSASClient()
		if err != nil {
			log.Fatalf("no client for SSAS; %s", err.Error())
		}
		return SSASPlugin{client: c, repository: repository}
	default:
		return AlphaAuthPlugin{}
	}
}

type AuthData struct {
	ACOID       string
	TokenID     string
	ClientID    string
	SystemID    string
	CMSID       string
	Blacklisted bool
}

type Credentials struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	ClientName   string    `json:"client_name"`
	SystemID     string    `json:"system_id"`
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Provider defines operations performed through an authentication provider.
type Provider interface {
	// RegisterSystem adds a software client for the ACO identified by localID.
	RegisterSystem(localID, publicKey, groupID string, ips ...string) (Credentials, error)

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

	// TODO refactor input to be AuthData
	// AuthorizeAccess asserts that a base64 encoded token string is valid for accessing the BCDA API
	AuthorizeAccess(tokenString string) error

	// VerifyToken decodes a base64 encoded token string into a structured token
	VerifyToken(tokenString string) (*jwt.Token, error)

	// GetVersion gets the version of the provider
	GetVersion() (string, error)
}
