package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/dgrijalva/jwt-go"
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
	p SSASPlugin
}

func (s *SSASPluginTestSuite) SetupSuite() {
	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf("no client for SSAS; %s", err.Error())
	}
	s.p = SSASPlugin{client: c}
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

func (s *SSASPluginTestSuite) TestResetSecret() {
	router := chi.NewRouter()
	router.Put("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, "fake-client-id", "fake-secret")
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf("no client for SSAS; %s", err.Error())
	}
	s.p = SSASPlugin{client: c}

	creds, err := s.p.ResetSecret("1")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "fake-client-id", creds.ClientID)
	assert.Equal(s.T(), "fake-secret", creds.ClientSecret)
}

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

	ts, err := s.p.MakeAccessToken(Credentials{ClientID: "happy", ClientSecret: "customer"})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), ts)
	assert.Regexp(s.T(), regexp.MustCompile(`[^.\s]+\.[^.\s]+\.[^.\s]+`), ts)

	ts, err = s.p.MakeAccessToken(Credentials{ClientID: "sad", ClientSecret: "customer"})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{ClientID: uuid.NewRandom().String()})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{ClientID: uuid.NewRandom().String(), ClientSecret: testUtils.RandomBase64(20)})
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

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf("no client for SSAS; %s", err.Error())
	}
	s.p = SSASPlugin{client: c}

	err = s.p.RevokeAccessToken("i.am.not.a.token")
	assert.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestAuthorizeAccess() {}

func (s *SSASPluginTestSuite) TestVerifyToken() {
	const fakeTokenString = "eyJhbGciOiJSUzI1NiIsIng1dCI6IjdkRC1nZWNOZ1gxWmY3R0xrT3ZwT0IyZGNWQSIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJodHRwczovL2NvbnRvc28uY29tIiwiaXNzIjoiaHR0cHM6Ly9zdHMud2luZG93cy5uZXQvZTQ4MTc0N2YtNWRhNy00NTM4LWNiYmUtNjdlNTdmN2QyMTRlLyIsIm5iZiI6MTM5MTIxMDg1MCwiZXhwIjoxMzkxMjE0NDUwLCJzdWIiOiIyMTc0OWRhYWUyYTkxMTM3YzI1OTE5MTYyMmZhMSJ9.C4Ny4LeVjEEEybcA1SVaFYFS6nH-Ezae_RrTXUYInjXGt-vBOkAa2ryb-kpOlzU_R4Ydce9tKDNp1qZTomXgHjl-cKybAz0Ut90-dlWgXGvJYFkWRXJ4J0JyS893EDwTEHYaAZH_lCBvoYPhXexD2yt1b-73xSP6oxVlc_sMvz3DY__1Y_OyvbYrThHnHglxvjh88x_lX7RN-Bq82ztumxy97rTWaa_1WJgYuy7h7okD24FtsD9PPLYAply0ygl31ReI0FZOdX12Hl4THJm4uI_4_bPXL6YR2oZhYWp-4POWIPHzG9c_GL8asBjoDY9F5q1ykQiotUBESoMML7_N1g"
	router := chi.NewRouter()
	router.Post("/introspect", func(w http.ResponseWriter, r *http.Request) {
		clientId, secret, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if clientId == os.Getenv("BCDA_SSAS_CLIENT_ID") && secret == os.Getenv("BCDA_SSAS_SECRET") {
			b, _ := json.Marshal(struct {
				Active bool `json:"active"`
			}{Active: true})
			if _, err := w.Write(b); err != nil {
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

	t, err := s.p.VerifyToken(fakeTokenString)
	assert.NotEmpty(s.T(), t)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &jwt.Token{}, t, "expected jwt token")
	c := t.Claims.(*CommonClaims)
	assert.Equal(s.T(), "21749daae2a91137c259191622fa1", c.Subject)
	assert.Equal(s.T(), int64(1391210850), c.NotBefore)
	assert.Equal(s.T(), int64(1391214450), c.ExpiresAt)
}

func TestSSASPluginSuite(t *testing.T) {
	suite.Run(t, new(SSASPluginTestSuite))
}
