package client_test

import (
	"encoding/json"
	"fmt"
	"io"
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

func (s *SSASClientTestSuite) SetupTest() {
	origSSASUseTLS = os.Getenv("SSAS_USE_TLS")
	origSSASURL = os.Getenv("SSAS_URL")
	origPublicURL = os.Getenv("SSAS_PUBLIC_URL")
	origBCDACAFile = os.Getenv("BCDA_CA_FILE")
	origSSASClientID = os.Getenv("BCDA_SSAS_CLIENT_ID")
	origSSASSecret = os.Getenv("BCDA_SSAS_SECRET")
}

func (s *SSASClientTestSuite) TearDownTest() {
	os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
	os.Setenv("SSAS_URL", origSSASURL)
	os.Setenv("SSAS_PUBLIC_URL", origPublicURL)
	os.Setenv("BCDA_SSAS_CLIENT_ID", origSSASClientID)
	os.Setenv("BCDA_SSAS_SECRET", origSSASSecret)
	os.Setenv("BCDA_CA_FILE", origBCDACAFile)
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalse() {
	os.Setenv("SSAS_USE_TLS", "false")
	os.Setenv("SSAS_URL", "http://ssas-url")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSTrue() {
	os.Setenv("SSAS_USE_TLS", "true")
	os.Setenv("SSAS_URL", "https://ssas-url")
	os.Setenv("BCDA_CA_FILE", "../../../shared_files/bcda.ca.pem")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoCertFile() {
	os.Setenv("SSAS_USE_TLS", "true")
	os.Setenv("SSAS_URL", "http://ssas-url")
	os.Unsetenv("BCDA_CA_FILE")

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: could not read CA file: read .: is a directory")
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoURL() {
	os.Unsetenv("SSAS_USE_TLS")
	os.Unsetenv("SSAS_URL")

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: no URL provided")
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalseNoURL() {
	os.Setenv("SSAS_USE_TLS", "false")
	os.Unsetenv("SSAS_URL")

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

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
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

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}

	resp, err := client.CreateSystem("fake-name", "fake-group", "fake-scope", "fake-key", "fake-tracking", nil)
	assert.Nil(s.T(), err)
	creds := auth.Credentials{}
	err = json.Unmarshal(resp, &creds)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "fake-client-id", creds.ClientID)
	assert.Equal(s.T(), "fake-secret", creds.ClientSecret)
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

func (s *SSASClientTestSuite) TestResetCredentials() {
	router := chi.NewRouter()
	router.Put("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, "fake-client-id", "fake-secret")
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}

	resp, err := client.ResetCredentials("1")
	assert.Nil(s.T(), err)
	creds := auth.Credentials{}
	err = json.Unmarshal(resp, &creds)
	assert.Nil(s.T(), err, nil)
	assert.Equal(s.T(), "fake-client-id", creds.ClientID)
	assert.Equal(s.T(), "fake-secret", creds.ClientSecret)
}

func (s *SSASClientTestSuite) TestDeleteCredentials() {
	router := chi.NewRouter()
	router.Delete("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

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

func (s *SSASClientTestSuite) TestRevokeAccessToken() {
	router := chi.NewRouter()
	router.Delete("/token/{tokenID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
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

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}

	respKey, err := client.GetToken(authclient.Credentials{ClientID: "happy", ClientSecret: "client"})
	if err != nil {
		s.FailNow("Failed to get token", err.Error())
	}

	assert.Equal(s.T(), tokenString, string(respKey))
}

func (s *SSASClientTestSuite) TestGetVersionPassing() {
	router := chi.NewRouter()
	router.Get("/_version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, "{\"version\":\"foo\"}\n")
		if err != nil {
			s.FailNow("Failed to create SSAS server", err.Error())
		}
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}
	version, err := client.GetVersion()
	assert.Equal(s.T(), "foo", version)
	assert.Nil(s.T(), err)
}

func (s *SSASClientTestSuite) TestGetVersionFailing() {
	router := chi.NewRouter()
	router.Get("/_version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, "{\"foo\":\"bar\"}\n")
		if err != nil {
			s.FailNow("Failed to create SSAS server", err.Error())
		}
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
	}
	version, err := client.GetVersion()
	assert.Equal(s.T(), "", version)
	assert.NotNil(s.T(), err)
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

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")
	os.Setenv("BCDA_SSAS_CLIENT_ID", "happy")
	os.Setenv("BCDA_SSAS_SECRET", "customer")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow("Failed to create SSAS client", err.Error())
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
