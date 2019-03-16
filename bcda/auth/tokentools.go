package auth

import (
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/utils"
)

var tokenTTL = time.Hour

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	setTokenDuration()
}

type CommonClaims struct {
	ClientID string   `json:"cid,omitempty"`
	Scopes   []string `json:"scp,omitempty"`
	ACOID    string   `json:"aco,omitempty"`
	UUID     string   `json:"id,omitempty"`
	jwt.StandardClaims
}

// TokenStringWithIDs generates a tokenstring that expires in tokenTTL time
func TokenStringWithIDs(tokenID, userID, acoID string) (string, error) {
	return TokenStringExpiration(tokenID, userID, acoID, tokenTTL)
}

// TokenStringExpiration generates a tokenstring that expires after a specific duration from now.
// If duration is <= 0, the token will be expired upon creation
func TokenStringExpiration(tokenID, userID, acoID string, duration time.Duration) (string, error) {
	return GenerateTokenString(tokenID, userID, acoID, time.Now().Unix(), time.Now().Add(duration).Unix())
}

// GenerateTokenString construct a token string for which all claims are specified in the call.
func GenerateTokenString(id, userID, acoID string, issuedAt int64, expiresAt int64) (string, error) {
	token := jwt.New(jwt.SigningMethodRS512)
	token.Claims = jwt.MapClaims{
		"exp": expiresAt,
		"iat": issuedAt,
		"sub": userID,
		"aco": acoID,
		"id":  id,
	}
	return token.SignedString(InitAlphaBackend().PrivateKey)
}

// for testing only; we don't support changing the ttl during runtime
func setTokenDuration() {
	if ttl := utils.FromEnv("JWT_EXPIRATION_DELTA", "60"); ttl != "" {
		var (
			n   int
			err error
		)
		if n, err = strconv.Atoi(ttl); err != nil {
			log.Infof("Invalid ttl %s in JWT_EXPIRATION_DELTA because %s; using %v", ttl, err, tokenTTL)
			return
		}
		tokenTTL = time.Minute * time.Duration(n)
	}
}
