package client_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"
)

var (
	origSSASURL      string
	origPublicURL    string
	origSSASUseTLS   string
	origSSASClientID string
	origSSASSecret   string
	origBCDACAFile   string
)

type SSASClientTestSuite struct {
	suite.Suite
}

const (
	ssasUseTLSKey            = "SSAS_USE_TLS"
	ssasURLKey               = "SSAS_URL"
	ssasPublicURLKey         = "SSAS_PUBLIC_URL"
	bcdaCAFileKey            = "BCDA_CA_FILE"
	bcdaSsasClientIDKey      = "BCDA_SSAS_CLIENT_ID"
	bcdaSsasSecret           = "BCDA_SSAS_SECRET"
	ssasClientFailureMessage = "Failed to create SSAS client"
	fakeClientId             = "fake-client-id"
	fakeSecret               = "fake-secret"
)

func (s *SSASClientTestSuite) SetupTest() {
	origSSASUseTLS = os.Getenv(ssasUseTLSKey)
	origSSASURL = os.Getenv(ssasURLKey)
	origPublicURL = os.Getenv(ssasPublicURLKey)
	origBCDACAFile = os.Getenv(bcdaCAFileKey)
	origSSASClientID = os.Getenv(bcdaSsasClientIDKey)
	origSSASSecret = os.Getenv(bcdaSsasSecret)
}

func (s *SSASClientTestSuite) TearDownTest() {
	os.Setenv(ssasUseTLSKey, origSSASUseTLS)
	os.Setenv(ssasURLKey, origSSASURL)
	os.Setenv(ssasPublicURLKey, origPublicURL)
	os.Setenv(bcdaSsasClientIDKey, origSSASClientID)
	os.Setenv(bcdaSsasSecret, origSSASSecret)
	os.Setenv(bcdaCAFileKey, origBCDACAFile)
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalse() {
	os.Setenv(ssasUseTLSKey, "false")
	os.Setenv(ssasURLKey, "http://ssas-url")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSTrue() {
	os.Setenv(ssasUseTLSKey, "true")
	os.Setenv(ssasURLKey, "https://ssas-url")
	os.Setenv(bcdaCAFileKey, "../../../shared_files/bcda.ca.pem")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoCertFile() {
	os.Setenv(ssasUseTLSKey, "true")
	os.Setenv(ssasURLKey, "http://ssas-url")
	os.Unsetenv(bcdaCAFileKey)

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: could not read CA file: read .: is a directory")
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoURL() {
	os.Unsetenv(ssasUseTLSKey)
	os.Unsetenv(ssasURLKey)

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: no URL provided")
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalseNoURL() {
	os.Setenv(ssasUseTLSKey, "false")
	os.Unsetenv(ssasURLKey)

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: no URL provided")
}

func (s *SSASClientTestSuite) TestCreateGroup() {
	router := chi.NewRouter()
	router.Post("/group", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{ "ID": 123456 }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	resp, err := client.CreateGroup("1", "name", "")
	if err != nil {
		s.FailNow("Failed to create group", err.Error())
	}

	assert.Equal(s.T(), `{ "ID": 123456 }`, string(resp))
}

func (s *SSASClientTestSuite) TestCreateSystem() {
	router := chi.NewRouter()
	router.Post("/system", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"system_id": "1", "client_id": "fake-client-id", "client_secret": "fake-secret", "client_name": "fake-name"}`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	resp, err := client.CreateSystem("fake-name", "fake-group", "fake-scope", "fake-key", "fake-tracking")
	assert.Nil(s.T(), err)
	creds := auth.Credentials{}
	err = json.Unmarshal(resp, &creds)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), fakeClientId, creds.ClientID)
	assert.Equal(s.T(), fakeSecret, creds.ClientSecret)
}

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

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	respKey, err := client.GetPublicKey(1)
	if err != nil {
		s.FailNow("Failed to get public key", err.Error())
	}

	assert.Equal(s.T(), keyStr, string(respKey))
}

func (s *SSASClientTestSuite) TestResetCredentials() {
	router := chi.NewRouter()
	router.Put("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, fakeClientId, fakeSecret)
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	resp, err := client.ResetCredentials("1")
	assert.Nil(s.T(), err)
	creds := auth.Credentials{}
	err = json.Unmarshal(resp, &creds)
	assert.Nil(s.T(), err, nil)
	assert.Equal(s.T(), fakeClientId, creds.ClientID)
	assert.Equal(s.T(), fakeSecret, creds.ClientSecret)
}

func (s *SSASClientTestSuite) TestDeleteCredentials() {
	router := chi.NewRouter()
	router.Delete("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	err = client.DeleteCredentials("1")
	assert.Nil(s.T(), err)
}

func (s *SSASClientTestSuite) TestRevokeAccessToken() {
	router := chi.NewRouter()
	router.Delete("/token/{tokenID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	err = client.RevokeAccessToken("abc-123")
	assert.Nil(s.T(), err)
}

func (s *SSASClientTestSuite) TestGetToken() {
	const tokenString = "totallyfake.tokenstringfor.testing"
	router := chi.NewRouter()
	router.Post("/token", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "token_type": "bearer", "access_token": "` + tokenString + `" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	respKey, err := client.GetToken(authclient.Credentials{ClientID: "happy", ClientSecret: "client"})
	if err != nil {
		s.FailNow("Failed to get token", err.Error())
	}

	assert.Equal(s.T(), tokenString, string(respKey))
}

func (s *SSASClientTestSuite) TestVerifyPublicToken() {
	const tokenString = "totallyfake.tokenstringfor.testing"
	router := chi.NewRouter()
	router.Post("/introspect", func(w http.ResponseWriter, r *http.Request) {
		var (
			buf   []byte
			input struct {
				Token string `json:"token"`
			}
		)
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			s.FailNow("unexpected failure %s", err.Error())
		}

		if err := json.Unmarshal(buf, &input); err != nil {
			s.FailNow("unexpected failure %s", err.Error())
		}

		body, err := json.Marshal(struct {
			Active bool `json:"active"`
		}{Active: true})
		if err != nil {
			s.FailNow("Invalid response in mock ssas server")
		}

		if _, err := w.Write(body); err != nil {
			s.FailNow("Write failure in mock ssas server; %s", err)
		}
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")
	os.Setenv(bcdaSsasClientIDKey, "happy")
	os.Setenv(bcdaSsasSecret, "customer")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(ssasClientFailureMessage, err.Error())
	}

	b, err := client.VerifyPublicToken(tokenString)
	if err != nil {
		s.FailNow("unexpected failure", err.Error())
	}

	var ir map[string]interface{}
	if err = json.Unmarshal(b, &ir); err != nil {
		s.FailNow("could not understand response", err.Error())
	}

	assert.True(s.T(), ir["active"].(bool))
}

func TestSSASClientTestSuite(t *testing.T) {
	suite.Run(t, new(SSASClientTestSuite))
}
