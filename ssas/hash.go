package ssas

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/CMSgov/bcda-app/ssas/cfg"
)

var (
	hashIter int
	hashKeyLen int
	saltSize int
)

// Hash is a cryptographically hashed string
type Hash string

// The time for hash comparison should be about 1s.  Increase hashIter if this is significantly faster in production.
// Note that changing hashIter or hashKeyLen will result in invalidating existing stored hashes (e.g. credentials).
func init() {
	hashIter = cfg.GetEnvInt("SSAS_HASH_ITERATIONS", 0)
	hashKeyLen = cfg.GetEnvInt("SSAS_HASH_KEY_LENGTH", 0)
	saltSize = cfg.GetEnvInt("SSAS_HASH_SALT_SIZE", 0)

	if hashIter == 0 || hashKeyLen == 0 || saltSize == 0 {
		// ServiceHalted(Event{Help:"SSAS_HASH_ITERATIONS, SSAS_HASH_KEY_LENGTH and SSAS_HASH_SALT_SIZE environment values must be set"})
		panic("SSAS_HASH_ITERATIONS, SSAS_HASH_KEY_LENGTH and SSAS_HASH_SALT_SIZE environment values must be set")
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
	hashEvent := Event{Elapsed: hashCreationTime}
	SecureHashTime(hashEvent)

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

