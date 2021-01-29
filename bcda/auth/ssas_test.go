package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	origSSASURL            string
	origPublicURL          string
	origSSASUseTLS         string
	origSSASClientKeyFile  string
	origSSASClientCertFile string
	origSSASClientID       string
	origSSASSecret         string
)

var testACOUUID = "dd0a19eb-c614-46c7-9ec0-95bbae959f37"

type SSASPluginTestSuite struct {
	suite.Suite
	p SSASPlugin
}

func (s *SSASPluginTestSuite) SetupSuite() {
	// original values must be saved before we run any tests that might change them
	origSSASUseTLS = os.Getenv("SSAS_USE_TLS")
	origSSASURL = os.Getenv("SSAS_URL")
	origPublicURL = os.Getenv("SSAS_PUBLIC_URL")
	origSSASClientKeyFile = os.Getenv("SSAS_CLIENT_KEY_FILE")
	origSSASClientCertFile = os.Getenv("SSAS_CLIENT_CERT_FILE")
	origSSASClientID = os.Getenv("BCDA_SSAS_CLIENT_ID")
	origSSASSecret = os.Getenv("BCDA_SSAS_SECRET")
}

func (s *SSASPluginTestSuite) SetupTest() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	db.Create(&models.ACO{
		UUID:     uuid.Parse(testACOUUID),
		Name:     "SSAS Plugin Test ACO",
		ClientID: testACOUUID,
	})
}

func (s *SSASPluginTestSuite) TearDownTest() {
	os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_URL", origSSASURL)
	os.Setenv("SSAS_PUBLIC_URL", origPublicURL)
	os.Setenv("SSAS_CLIENT_KEY_FILE", origSSASClientKeyFile)
	os.Setenv("SSAS_CLIENT_CERT_FILE", origSSASClientCertFile)
	os.Setenv("BCDA_SSAS_CLIENT_ID", origSSASClientID)
	os.Setenv("BCDA_SSAS_SECRET", origSSASSecret)

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	db.Unscoped().Delete(models.ACO{}, "uuid = ?", testACOUUID)
}

func (s *SSASPluginTestSuite) TestRegisterSystem() {
	// These variables will allow us to swap out expectations without
	// reinstantiating the server
	var (
		response string
		ips      []string
		tester   *testing.T
	)

	// TODO: Mock client instead of server
	router := chi.NewRouter()
	router.Post("/system", func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		assert.NoError(tester, err)
		var obj map[string]interface{}
		assert.NoError(tester, json.Unmarshal(reqBody, &obj))
		if obj["ips"] == nil {
			assert.Equal(tester, 0, len(ips), "ips should be empty since request contained no ips field")
		} else {
			var ipsReceived []string
			for _, ip := range obj["ips"].([]interface{}) {
				ipsReceived = append(ipsReceived, ip.(string))
			}
			assert.Equal(tester, ipsReceived, ips)
		}

		w.WriteHeader(201)
		fmt.Fprint(w, response)
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

	validResp := `{ "system_id": "1", "client_id": "fake-client-id", "client_secret": "fake-secret", "client_name": "fake-name" }`
	tests := []struct {
		name      string
		ips       []string
		ssasResp  string
		expErrMsg string
	}{
		{"Successful response", nil, validResp, ""},
		{"Successful response with IPs", []string{testUtils.GetRandomIPV4Address(s.T()), testUtils.GetRandomIPV4Address(s.T())}, validResp, ""},
		{"Invalid JSON response", nil, `"this is": "invalid"`, "failed to unmarshal response json"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			ips, response, tester = tt.ips, tt.ssasResp, t
			creds, err := s.p.RegisterSystem(testACOUUID, "", "", tt.ips...)
			if tt.expErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to unmarshal response json")
				assert.Empty(t, creds.SystemID)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "1", creds.SystemID)
			assert.Equal(t, "fake-client-id", creds.ClientID)
		})
	}
}

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

	creds, err := s.p.ResetSecret(testACOUUID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "fake-client-id", creds.ClientID)
	assert.Equal(s.T(), "fake-secret", creds.ClientSecret)
}

func (s *SSASPluginTestSuite) TestRevokeSystemCredentials() {}

func (s *SSASPluginTestSuite) TestMakeAccessToken() {
	_, tokenString, _ := MockSSASToken()
	MockSSASServer(tokenString)

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf("no client for SSAS; %s", err.Error())
	}
	s.p = SSASPlugin{client: c}

	ts, err := s.p.MakeAccessToken(Credentials{ClientID: "mock-client", ClientSecret: "mock-secret"})
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

func (s *SSASPluginTestSuite) TestAuthorizeAccess() {
	_, ts, err := MockSSASToken()
	require.NotNil(s.T(), ts, "no token for SSAS", err)
	require.Nil(s.T(), err, "unexpected error; ", err)
	MockSSASServer(ts)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, "no client for SSAS; ", err)
	s.p = SSASPlugin{client: c}
	err = s.p.AuthorizeAccess(ts)
	require.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestVerifyToken() {
	_, ts, err := MockSSASToken()
	require.NotNil(s.T(), ts, "no token for SSAS; ", err)
	require.Nil(s.T(), err, "unexpected error; ", err)
	MockSSASServer(ts)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, "no client for SSAS; ")
	require.Nil(s.T(), err, "unexpected error; ", err)
	s.p = SSASPlugin{client: c}

	t, err := s.p.VerifyToken(ts)
	assert.NotEmpty(s.T(), t)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &jwt.Token{}, t, "expected jwt token")
	assert.True(s.T(), t.Valid)
	tc := t.Claims.(*CommonClaims)
	assert.Equal(s.T(), "mock-system", tc.SystemID)
	assert.Equal(s.T(), "mock-id", tc.Id)
}

func TestSSASPluginSuite(t *testing.T) {
	suite.Run(t, new(SSASPluginTestSuite))
}

func MockSSASServer(tokenString string) {
	os.Setenv("BCDA_SSAS_CLIENT_ID", "bcda")
	os.Setenv("BCDA_SSAS_SECRET", "api")
	router := chi.NewRouter()
	router.Post("/introspect", func(w http.ResponseWriter, r *http.Request) {
		clientId, secret, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var answer = make(map[string]bool)
		if clientId == os.Getenv("BCDA_SSAS_CLIENT_ID") && secret == os.Getenv("BCDA_SSAS_SECRET") {
			answer["active"] = true
		} else {
			answer["active"] = false
		}
		render.JSON(w, r, answer)
	})
	router.Post("/token", func(w http.ResponseWriter, r *http.Request) {
		clientId, _, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if clientId == "mock-client" {
			render.JSON(w, r, client.TokenResponse{AccessToken: tokenString, TokenType: "access_token"})
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})

	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")
}

func MockSSASToken() (*jwt.Token, string, error) {
	// NB: currently, BCDA expects only 1 item in the array of cms_ids. At some point, ACO-MS will want to send more than one
	claims := CommonClaims{
		SystemID: "mock-system",
		Data:     `{"cms_ids":["A9995"]}`,
		ClientID: "mock-client",
		StandardClaims: jwt.StandardClaims{
			Issuer:    "ssas",
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Id:        "mock-id",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	ts, err := InitAlphaBackend().SignJwtToken(t)
	return t, ts, err
}
