package auth_test

import (
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/stretchr/testify/suite"
)

var (
	origSSASURL				string
	origPublicURL			string
	origSSASUseTLS			string
	origSSASClientKeyFile	string
	origSSASClientCertFile	string
)

type SSASPluginTestSuite struct {
	suite.Suite
	p auth.SSASPlugin
}

func (s *SSASPluginTestSuite) SetupSuite() {
	s.p = auth.SSASPlugin{}
}

func (s *SSASPluginTestSuite) BeforeTest() {
	origSSASUseTLS = os.Getenv("SSAS_USE_TLS")
	origSSASURL = os.Getenv("SSAS_URL")
	origPublicURL = os.Getenv("SSAS_PUBLIC_URL")
	origSSASClientKeyFile = os.Getenv("SSAS_CLIENT_KEY_FILE")
	origSSASClientCertFile = os.Getenv("SSAS_CLIENT_CERT_FILE")
}

func (s *SSASPluginTestSuite) AfterTest() {
	os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_URL", origSSASURL)
	os.Setenv("SSAS_PUBLIC_URL", origPublicURL)
	os.Setenv("SSAS_CLIENT_KEY_FILE", origSSASClientKeyFile)
	os.Setenv("SSAS_CLIENT_CERT_FILE", origSSASClientCertFile)
}

func (s *SSASPluginTestSuite) TestRegisterSystem() {}

func (s *SSASPluginTestSuite) TestUpdateSystem() {}

func (s *SSASPluginTestSuite) TestDeleteSystem() {}

func (s *SSASPluginTestSuite) TestResetSecret() {}

func (s *SSASPluginTestSuite) TestMakeAccessToken() {}

func (s *SSASPluginTestSuite) TestRevokeAccessToken() {
	router := chi.NewRouter()
	router.Delete("/token/{tokenID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	err := s.p.RevokeAccessToken("i.am.not.a.token")
	assert.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestAuthorizeAccess() {}

func (s *SSASPluginTestSuite) TestVerifyToken() {}

func (s *SSASPluginTestSuite) TestRevokeSystemCredentials() {}

func TestSSASPluginSuite(t *testing.T) {
	suite.Run(t, new(SSASPluginTestSuite))
}
