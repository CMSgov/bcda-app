package auth

import (
	"crypto/rsa"
	"os"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost int = 10

var (
	alphaBackend *AlphaBackend
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

// Hash is a cryptographically hashed string
type Hash string

// NewHash creates a hashed string from a source string, return it as a new Hash value.
func NewHash(source string) (Hash, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(source), bcryptCost)
	if err != nil {
		// Could this lead to downstream logging of secrets?  A review of the source says no.  The most likely error seems to be
		//   an invalid cost.
		return "", err
	}
	return Hash(string(hash)), nil
}

// IsHashOf accepts an unhashed string, which it first hashes and then compares to itself
func (h Hash) IsHashOf(source string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(h), []byte(source))
	// Noted that this swallows any errors by returning false, which seems appropriate in the context.
	return err == nil
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
