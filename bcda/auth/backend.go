package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/secutils"
)

var (
	jwtExpirationDelta  string        = os.Getenv("JWT_EXPIRATION_DELTA")
	authBackendInstance *AlphaBackend = nil
)

type Hash struct{}

func (c *Hash) Generate(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func (c *Hash) Compare(hash string, s string) bool {
	return hash == c.Generate(s)
}

type AlphaBackend struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func InitAuthBackend() *AlphaBackend {
	if authBackendInstance == nil {
		authBackendInstance = &AlphaBackend{
			PrivateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}

	return authBackendInstance
}

// For testing.  Probably no real use case.
func (backend *AlphaBackend) ResetAuthBackend() {

	authBackendInstance = &AlphaBackend{
		PrivateKey: getPrivateKey(),
		PublicKey:  getPublicKey(),
	}
}

// This method and its sibling, getPublicKey(), get the private key from the file system and environment variables.
// They accesses external resources and so may panic and bubble up an error if the file is not present or not readable.
func getPrivateKey() *rsa.PrivateKey {
	privateKeyFile, err := os.Open(os.Getenv("JWT_PRIVATE_KEY_FILE"))
	if err != nil {
		log.Panic(err)
	}
	return secutils.OpenPrivateKeyFile(privateKeyFile)
}

// panics if file is not found, corrupted, or otherwise unreadable
func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open(os.Getenv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		panic(err)
	}
	return secutils.OpenPublicKeyFile(publicKeyFile)
}

// Sign a prepared JWT token, returning it as a base-64 encoded string suitable for use as a Bearer token.
func (backend *AlphaBackend) SignJwtToken(token jwt.Token) (string, error) {
	return token.SignedString(backend.PrivateKey)
}
