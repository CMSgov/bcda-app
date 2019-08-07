package client_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"
)

type SSASClientTestSuite struct {
	suite.Suite
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalse() {
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	defer os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_USE_TLS", "false")

	origSSASURL := os.Getenv("SSAS_URL")
	defer os.Setenv("SSAS_URL", origSSASURL)
	os.Setenv("SSAS_URL", "http://ssas-url")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoKeypair() {
	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: could not load SSAS keypair: open : no such file or directory")
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalseNoURL() {
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	defer os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_USE_TLS", "false")

	origSSASURL := os.Getenv("SSAS_URL")
	defer os.Setenv("SSAS_URL", origSSASURL)
	os.Unsetenv("SSAS_URL")

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: no URL provided")
}

func (s *SSASClientTestSuite) TestCreateSystem() {}

func (s *SSASClientTestSuite) TestGetPublicKey() {
	router := chi.NewRouter()
	keyStr := "123456"
	router.Get("/system/{systemID}/key", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "client_id": "123456", "public_key": "` + keyStr + `" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	origSSASURL := os.Getenv("SSAS_URL")
	defer os.Setenv("SSAS_URL", origSSASURL)
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	defer os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}

	respKey, err := client.GetPublicKey(1)
	if err != nil {
		s.FailNow("Failed to get public key", err.Error())
	}

	assert.Equal(s.T(), keyStr, string(respKey))
}

func (s *SSASClientTestSuite) TestResetCredentials() {}

func (s *SSASClientTestSuite) TestDeleteCredentials() {
	router := chi.NewRouter()
	router.Delete("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	origSSASURL := os.Getenv("SSAS_URL")
	defer os.Setenv("SSAS_URL", origSSASURL)
	origPublicURL := os.Getenv("SSAS_PUBLIC_URL")
	defer os.Setenv("SSAS_PUBLIC_URL", origPublicURL)
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	defer os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}

	err = client.DeleteCredentials("1")
	assert.Nil(s.T(), err)
}

func TestSSASClientTestSuite(t *testing.T) {
	suite.Run(t, new(SSASClientTestSuite))
}
