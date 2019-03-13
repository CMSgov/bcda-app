package auth

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

type CommonClaims struct {
	ClientID string   `json:"cid,omitempty"`
	Scopes   []string `json:"scp,omitempty"`
	ACOID    string   `json:"aco,omitempty"`
	UUID     string   `json:"id,omitempty"`
	jwt.StandardClaims
}

// Given all required ids, generate a tokenstring that expires in one hour
func TokenStringWithIDs(tokenID, userID, acoID string) (string, error) {
	return TokenStringExpiration(tokenID, userID, acoID, time.Hour)
}

// Given all required ids and a duration, generate a tokenstring that expires duration from now.
// If duration is <= 0, the token will be expired upon creation
func TokenStringExpiration(tokenID, userID, acoID string, duration time.Duration) (string, error) {
	return GenerateTokenString(tokenID, userID, acoID, time.Now().Unix(), time.Now().Add(duration).Unix())
}

// Given all alpha claim values, construct a token string.
func GenerateTokenString(id, userID, acoID string, issuedAt int64, expiresAt int64) (string, error) {
	token := jwt.New(jwt.SigningMethodRS512)
	token.Claims = jwt.MapClaims{
		"exp": expiresAt,
		"iat": issuedAt,
		"sub": userID,
		"aco": acoID,
		"id":  id,
	}
	return token.SignedString(InitAuthBackend().PrivateKey)
}
