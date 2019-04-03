package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/utils"
)

var (
	alphaBackend *AlphaBackend
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

// Hash is a cryptographically hashed string
type Hash string

// NewHash creates a hashed string from a source string, return it as a new Hash value.
func NewHash(source string) Hash {
	sum := sha256.Sum256([]byte(source))
	return Hash(fmt.Sprintf("%x", sum))
}

// IsHashOf accepts an unhashed string, which it first hashes and then compares to itself
func (h Hash) IsHashOf(source string) bool {
	return h == NewHash(source)
}

func (h Hash) String() string {
	return string(h)
}

// AlphaBackend is the authorization backend for the alpha plugin. Its purpose is to hold and control use of the
// server's public and private keys.
type AlphaBackend struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// InitAlphaBackend does first time initialization of the alphaBackend instance with its private and public key pair.
// If the instance is already initialized, it simply returns the existing value.
func InitAlphaBackend() *AlphaBackend {
	if alphaBackend == nil {
		alphaBackend = &AlphaBackend{
			PrivateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}
	return alphaBackend
}

// ResetAlphaBackend sets the servers keys whether they are set or not. Used for testing.
func (backend *AlphaBackend) ResetAlphaBackend() {
	alphaBackend = &AlphaBackend{
		PrivateKey: getPrivateKey(),
		PublicKey:  getPublicKey(),
	}
}

// This method and its sibling, getPublicKey(), get the private key from the file system and environment variables.
// They accesses external resources and so may panic and bubble up an error if the file is not present or not readable.
func getPrivateKey() *rsa.PrivateKey {
	fileName, ok := os.LookupEnv("JWT_PRIVATE_KEY_FILE")
	if !ok {
		log.Panic("no value in JWT_PRIVATE_KEY_FILE")
	}
	log.Infof("opening %s", fileName)
	/* #nosec -- Potential file inclusion via variable */
	privateKeyFile, err := os.Open(fileName)
	if err != nil {
		log.Panicf("can't open private key file %s because %v", fileName, err)
	}

	return utils.OpenPrivateKeyFile(privateKeyFile)
}

// panics if file is not found, corrupted, or otherwise unreadable
func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open(os.Getenv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		panic(err)
	}
	return utils.OpenPublicKeyFile(publicKeyFile)
}

// SignJwtToken signs a prepared JWT token, returning it as a base-64 encoded string suitable for use as a Bearer token.
func (backend *AlphaBackend) SignJwtToken(token jwt.Token) (string, error) {
	return token.SignedString(backend.PrivateKey)
}
