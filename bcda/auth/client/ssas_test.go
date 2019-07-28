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

func (s *SSASClientTestSuite) TestNewSSASClient() {}

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

func (s *SSASClientTestSuite) TestDeleteCredentials() {}

func TestSSASClientTestSuite(t *testing.T) {
	suite.Run(t, new(SSASClientTestSuite))
}
