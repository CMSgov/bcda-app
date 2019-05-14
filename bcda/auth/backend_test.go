package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type BackendTestSuite struct {
	suite.Suite
	AuthBackend   *auth.AlphaBackend
	TmpFiles      []string
	expectedSizes map[string]int
}

func (s *BackendTestSuite) SetupSuite() {
	models.InitializeGormModels()
	auth.InitializeGormModels()
}

func (s *BackendTestSuite) CreateTempFile() (*os.File, error) {
	tmpfile, err := ioutil.TempFile("", "bcda_backend_test_")
	if err != nil {
		return &os.File{}, err
	}

	return tmpfile, nil
}

func (s *BackendTestSuite) SavePrivateKey(f *os.File, key *rsa.PrivateKey) {
	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err := pem.Encode(f, privateKey)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *BackendTestSuite) SavePubKey(f *os.File, pubkey rsa.PublicKey) {
	asn1Bytes, err := x509.MarshalPKIXPublicKey(&pubkey)
	if err != nil {
		log.Fatal(err)
	}

	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	err = pem.Encode(f, pemkey)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *BackendTestSuite) SetupAuthBackend() {
	reader := rand.Reader
	bitSize := 1024

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		log.Fatal(err)
	}
	publicKey := key.PublicKey

	privKeyFile, err := s.CreateTempFile()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Setenv("JWT_PRIVATE_KEY_FILE", privKeyFile.Name())
	if err != nil {
		log.Panic(err)
	}
	s.TmpFiles = append(s.TmpFiles, privKeyFile.Name())
	s.SavePrivateKey(privKeyFile, key)
	defer privKeyFile.Close()

	pubKeyFile, err := s.CreateTempFile()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Setenv("JWT_PUBLIC_KEY_FILE", pubKeyFile.Name())
	if err != nil {
		log.Panic(err)
	}
	s.TmpFiles = append(s.TmpFiles, pubKeyFile.Name())
	s.SavePubKey(pubKeyFile, publicKey)
	defer pubKeyFile.Close()

	s.AuthBackend = auth.InitAlphaBackend()
}

func (s *BackendTestSuite) SetupTest() {
	s.SetupAuthBackend()
	s.expectedSizes = map[string]int{
		"dev":    50,
		"small":  10,
		"medium": 25,
		"large":  100,
	}
}

func (s *BackendTestSuite) TestInitAuthBackend() {
	assert.IsType(s.T(), &auth.AlphaBackend{}, s.AuthBackend)
	assert.IsType(s.T(), &rsa.PrivateKey{}, s.AuthBackend.PrivateKey)
	assert.IsType(s.T(), &rsa.PublicKey{}, s.AuthBackend.PublicKey)
}

func (s *BackendTestSuite) TestHashComparable() {
	uuidString := uuid.NewRandom().String()
	hash, err := auth.NewHash(uuidString)
	assert.Nil(s.T(), err)
	assert.True(s.T(), hash.IsHashOf(uuidString))
	assert.False(s.T(), hash.IsHashOf(uuid.NewRandom().String()))
}

func (s *BackendTestSuite) TestHashUnique() {
	uuidString := uuid.NewRandom().String()
	hash1, _ := auth.NewHash(uuidString)
	hash2, _ := auth.NewHash(uuidString)
	assert.NotEqual(s.T(), hash1.String(), hash2.String())
}

func (s *BackendTestSuite) TestHashCompatibility() {
	uuidString := "96c5a0cd-b284-47ac-be6e-f33b14dc4697"
	hash := auth.Hash("d3H4fX/uEk1jOW2gYrFezyuJoSv4ay2x3gH5C25KpWM=:kVqFm1he5S4R1/10oIkVNFot40VB3wTa+DXTp4TrwvyXHkQO7Dxjjo/OqwemiYP8p3UQ8r/HkmTQrSS99UXzaQ==")
	assert.True(s.T(), hash.IsHashOf(uuidString), "Possible change in hashing parameters or algorithm.  Known input/output does not match.  Merging this code will result in invalidating credentials.")
}

func (s *BackendTestSuite) TestHashEmpty() {
	hash, err := auth.NewHash("")
	assert.NotNil(s.T(), err)
	assert.False(s.T(), hash.IsHashOf(""))
}

func (s *BackendTestSuite) TestHashInvalid() {
	hash := auth.Hash("INVALID_NUMBER_OF_SEGMENTS:d3H4fX/uEk1jOW2gYrFezyuJoSv4ay2x3gH5C25KpWM=:kVqFm1he5S4R1/10oIkVNFot40VB3wTa+DXTp4TrwvyXHkQO7Dxjjo/OqwemiYP8p3UQ8r/HkmTQrSS99UXzaQ==")
	assert.False(s.T(), hash.IsHashOf("96c5a0cd-b284-47ac-be6e-f33b14dc4697"))
}

func (s *BackendTestSuite) TestPrivateKey() {
	privateKey := s.AuthBackend.PrivateKey
	assert.NotNil(s.T(), privateKey)
	// get the real Key File location
	actualPrivateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	defer func() { os.Setenv("JWT_PRIVATE_KEY_FILE", actualPrivateKeyFile) }()

	// set the Private Key File to a bogus value to test negative scenarios
	// File does not exist
	os.Setenv("JWT_PRIVATE_KEY_FILE", "/static/thisDoesNotExist.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// Empty file
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// File contains not a key
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/badPrivate.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)
}

func (s *BackendTestSuite) TestPublicKey() {
	privateKey := s.AuthBackend.PublicKey
	assert.NotNil(s.T(), privateKey)
	// get the real Key File location
	actualPublicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	defer func() { os.Setenv("JWT_PUBLIC_KEY_FILE", actualPublicKeyFile) }()

	// set the Private Key File to a bogus value to test negative scenarios
	// File does not exist
	os.Setenv("JWT_PUBLIC_KEY_FILE", "/static/thisDoesNotExist.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// Empty file
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// File contains not a key
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/badPublic.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)
}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
