package auth

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
)

const (
	SSAS = "ssas"
)

var providerName = SSAS
var repository models.Repository
var provider Provider

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)

	repository = postgres.NewRepository(database.Connection)

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf("no client for SSAS; %s", err.Error())
	}
	provider = SSASPlugin{client: c, repository: repository}

}

func GetProviderName() string {
	return providerName
}

func GetProvider() Provider {
	return provider
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

type CommonClaims struct {
	ClientID string   `json:"cid,omitempty"`
	SystemID string   `json:"sys,omitempty"`
	Data     string   `json:"dat,omitempty"`
	Scopes   []string `json:"scp,omitempty"`
	ACOID    string   `json:"aco,omitempty"`
	UUID     string   `json:"id,omitempty"`
	jwt.StandardClaims
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
