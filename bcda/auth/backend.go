package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/pbkdf2"

	"github.com/CMSgov/bcda-app/bcda/utils"
    configuration "github.com/CMSgov/bcda-app/config"
)

var (
	alphaBackend *AlphaBackend
	hashIter     int
	hashKeyLen   int
	saltSize     int
)

// Hash is a cryptographically hashed string
type Hash string

// The time for hash comparison should be about 1s.  Increase hashIter if this is significantly faster in production.
// Note that changing hashIter or hashKeyLen will result in invalidating existing stored hashes (e.g. credentials).
func init() {
	hashIter = utils.GetEnvInt("AUTH_HASH_ITERATIONS", 0)
	hashKeyLen = utils.GetEnvInt("AUTH_HASH_KEY_LENGTH", 0)
	saltSize = utils.GetEnvInt("AUTH_HASH_SALT_SIZE", 0)

	if hashIter == 0 || hashKeyLen == 0 || saltSize == 0 {
		serviceHalted(event{help: "HASH_ITERATIONS, HASH_KEY_LENGTH and HASH_SALT_SIZE environment values must be set"})
		panic("HASH_ITERATIONS, HASH_KEY_LENGTH and HASH_SALT_SIZE environment values must be set")
	}
}

// NewHash creates a Hash value from a source string
// The HashValue consists of the salt and hash separated by a colon ( : )
// If the source of randomness fails it returns an error.
func NewHash(source string) (Hash, error) {
	if source == "" {
		return Hash(""), errors.New("empty string provided to hash function")
	}

	salt := make([]byte, saltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return Hash(""), err
	}

	start := time.Now()
	h := pbkdf2.Key([]byte(source), salt, hashIter, hashKeyLen, sha512.New)
	hashCreationTime := time.Since(start)
	hashEvent := event{elapsed: hashCreationTime}
	secureHashTime(hashEvent)

	return Hash(fmt.Sprintf("%s:%s", base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(h))), nil
}

// IsHashOf accepts an unhashed string, which it first hashes and then compares to itself
func (h Hash) IsHashOf(source string) bool {
	// Avoid comparing with an empty source so that a hash of an empty string is never successful
	if source == "" {
		return false
	}

	hashAndPass := strings.Split(h.String(), ":")
	if len(hashAndPass) != 2 {
		return false
	}

	hash := hashAndPass[1]
	salt, err := base64.StdEncoding.DecodeString(hashAndPass[0])
	if err != nil {
		return false
	}

	sourceHash := pbkdf2.Key([]byte(source), salt, hashIter, hashKeyLen, sha512.New)
	return hash == base64.StdEncoding.EncodeToString(sourceHash)
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
		serviceStarted(event{})
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
		serviceHalted(event{help: "no value in JWT_PRIVATE_KEY_FILE"})
		panic(errors.New("no value in JWT_PRIVATE_KEY_FILE"))
	}
	logger.Infof("opening %s", fileName)
	/* #nosec -- Potential file inclusion via variable */
	privateKeyFile, err := os.Open(fileName)
	if err != nil {
		msg := fmt.Sprintf("can't open private key file %s because %v", fileName, err)
		serviceHalted(event{help: msg})
		panic(errors.New(msg))
	}

	return utils.OpenPrivateKeyFile(privateKeyFile)
}

// panics if file is not found, corrupted, or otherwise unreadable
func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open(configuration.GetEnv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		serviceHalted(event{help: err.Error()})
		panic(err)
	}
	return utils.OpenPublicKeyFile(publicKeyFile)
}

// SignJwtToken signs a prepared JWT token, returning it as a base-64 encoded string suitable for use as a Bearer token.
func (backend *AlphaBackend) SignJwtToken(token *jwt.Token) (string, error) {
	return token.SignedString(backend.PrivateKey)
}
