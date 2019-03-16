package testUtils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	suite.Suite
	AuthBackend *auth.AlphaBackend
	TmpFiles    []string
}

func PrintSeparator() {
	fmt.Println("**********************************************************************************")
}
func (s *AuthTestSuite) CreateTempFile() (*os.File, error) {
	tmpfile, err := ioutil.TempFile("", "bcda_backend_test_")
	if err != nil {
		return &os.File{}, err
	}

	return tmpfile, nil
}

func (s *AuthTestSuite) SavePrivateKey(f *os.File, key *rsa.PrivateKey) {
	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err := pem.Encode(f, privateKey)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *AuthTestSuite) SavePubKey(f *os.File, pubkey rsa.PublicKey) {
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

func (s *AuthTestSuite) SetupAuthBackend() {
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

	s.AuthBackend = auth.InitAuthBackend()
}

func CreateStaging(jobID string) {
	err := os.Setenv("FHIR_STAGING_DIR", "data/test")
	if err != nil {
		log.Panic(err)
	}
	testdir := fmt.Sprintf("%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID)

	if _, err := os.Stat(testdir); os.IsNotExist(err) {
		err = os.MkdirAll(testdir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func RandomHexID() string {
	b, err := someRandomBytes(4)
	if err != nil {
		return "not_a_random_client_id"
	}
	return fmt.Sprintf("%x", b)
}

func someRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}