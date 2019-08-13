package auth_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/testUtils"

	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	origSSASURL            string
	origPublicURL          string
	origSSASUseTLS         string
	origSSASClientKeyFile  string
	origSSASClientCertFile string
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

func (s *SSASPluginTestSuite) TestRevokeSystemCredentials() {}

func (s *SSASPluginTestSuite) TestMakeAccessToken() {
	const tokenString = "totallyfake.tokenstringfor.testing"
	router := chi.NewRouter()
	router.Post("/token", func(w http.ResponseWriter, r *http.Request) {
		clientId, secret, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if clientId == "happy" && secret == "customer" {
			_, err := w.Write([]byte(`{ "token_type": "bearer", "access_token": "` + tokenString + `" }`))
			if err != nil {
				log.Fatal(err)
			}
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	ts, err := s.p.MakeAccessToken(auth.Credentials{ClientID: "happy", ClientSecret: "customer"})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), ts)
	assert.Regexp(s.T(), regexp.MustCompile(`[^.\s]+\.[^.\s]+\.[^.\s]+`), ts)

	// t, _ := s.p.VerifyToken(ts)
	// c := t.Claims.(*auth.CommonClaims)
	// assert.True(s.T(), c.ExpiresAt <= time.Now().Unix()+3600)

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientID: "sad", ClientSecret: "customer"})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(auth.Credentials{})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientID: uuid.NewRandom().String()})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientID: uuid.NewRandom().String(), ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")
}

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

func TestSSASPluginSuite(t *testing.T) {
	suite.Run(t, new(SSASPluginTestSuite))
}
