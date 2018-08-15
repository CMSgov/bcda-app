package auth

import (
	"bufio"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
)

var (
	jwtExpirationDelta  string                    = os.Getenv("JWT_EXPIRATION_DELTA")
	authBackendInstance *JWTAuthenticationBackend = nil
)

type Hash struct{}

func (c *Hash) Generate(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func (c *Hash) Compare(hash string, s string) bool {
	return hash == c.Generate(s)
}

type JWTAuthenticationBackend struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func InitAuthBackend() *JWTAuthenticationBackend {
	if authBackendInstance == nil {
		authBackendInstance = &JWTAuthenticationBackend{
			PrivateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}

	return authBackendInstance
}

func (backend *JWTAuthenticationBackend) CreateACO(name string) (uuid.UUID, error) {
	db := database.GetDbConnection()
	defer db.Close()

	const sqlstr = `INSERT INTO public.acos (` +
		`uuid, name, created_at, updated_at` +
		`) VALUES (` +
		`$1, $2, $3, $3` +
		`)`

	acoUUID := uuid.NewUUID()
	now := time.Now()

	_, err := db.Exec(sqlstr, acoUUID, name, now)

	if err != nil {
		return nil, err
	}

	return acoUUID, nil
}

func (backend *JWTAuthenticationBackend) GenerateToken(userID string, acoID string) (string, error) {
	expirationDelta, err := strconv.Atoi(jwtExpirationDelta)
	if err != nil {
		expirationDelta = 72
	}

	token := jwt.New(jwt.SigningMethodRS512)
	token.Claims = jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * time.Duration(expirationDelta)).Unix(),
		"iat": time.Now().Unix(),
		"sub": userID,
		"aco": acoID,
	}
	tokenString, err := token.SignedString(backend.PrivateKey)
	if err != nil {
		panic(err)
	}
	return tokenString, nil
}

func (backend *JWTAuthenticationBackend) RevokeToken(tokenString string) bool {
	claims := getJWTClaims(backend, tokenString)

	if claims == nil {
		panic("Could not read token claims")
	}

	userID := claims["sub"].(string)

	token := models.Token{
		UserID: uuid.Parse(userID),
		Value:  tokenString,
		Active: false,
	}

	db := database.GetDbConnection()
	defer db.Close()

	err := token.Insert(db)
	if err != nil {
		panic(err)
	}

	return true
}

func (backend *JWTAuthenticationBackend) IsBlacklisted(token *jwt.Token) bool {
	var (
		err  error
		hash Hash = Hash{}
	)

	claims, _ := token.Claims.(jwt.MapClaims)

	db := database.GetDbConnection()
	defer db.Close()

	const sqlstr = `SELECT value ` +
		`FROM public.tokens ` +
		`WHERE user_id = $1 ` +
		`AND active = false`

	rows, err := db.Query(sqlstr, claims["sub"])
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		t := models.Token{}
		err = rows.Scan(&t.Value)
		if err != nil {
			log.Fatal(err)
		}

		if match := hash.Compare(t.Value, token.Raw); match {
			return true
		}
	}

	return false
}

func getPrivateKey() *rsa.PrivateKey {
	privateKeyFile, err := os.Open(os.Getenv("JWT_PRIVATE_KEY_FILE"))
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := privateKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)
	if err != nil {
		log.Fatal(err)
	}

	data, _ := pem.Decode([]byte(pembytes))

	privateKeyFile.Close()

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	return privateKeyImported
}

func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open(os.Getenv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := publicKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pembytes)
	if err != nil {
		log.Fatal(err)
	}

	data, _ := pem.Decode([]byte(pembytes))

	publicKeyFile.Close()

	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)

	if !ok {
		panic(err)
	}

	return rsaPub
}

func getJWTClaims(backend *JWTAuthenticationBackend, tokenString string) jwt.MapClaims {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return backend.PublicKey, nil
	})

	if err != nil {
		panic(err)
	}

	if !token.Valid {
		return nil
	}

	return token.Claims.(jwt.MapClaims)
}
