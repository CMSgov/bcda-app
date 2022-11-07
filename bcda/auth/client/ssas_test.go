package client_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"

	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
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
	origSSASUseTLS = conf.GetEnv("SSAS_USE_TLS")
	origSSASURL = conf.GetEnv("SSAS_URL")
	origPublicURL = conf.GetEnv("SSAS_PUBLIC_URL")
	origBCDACAFile = conf.GetEnv("BCDA_CA_FILE")
	origSSASClientID = conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	origSSASSecret = conf.GetEnv("BCDA_SSAS_SECRET")
}

func (s *SSASClientTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", origSSASUseTLS)
	conf.SetEnv(s.T(), "SSAS_URL", origSSASURL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", origPublicURL)
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", origSSASClientID)
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", origSSASSecret)
	conf.SetEnv(s.T(), "BCDA_CA_FILE", origBCDACAFile)
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalse() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "SSAS_URL", "http://ssas-url")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSTrue() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "true")
	conf.SetEnv(s.T(), "SSAS_URL", "https://ssas-url")
	conf.SetEnv(s.T(), "BCDA_CA_FILE", "../../../shared_files/bcda.ca.pem")

	client, err := authclient.NewSSASClient()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), client)
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoCertFile() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "true")
	conf.SetEnv(s.T(), "SSAS_URL", "http://ssas-url")
	conf.UnsetEnv(s.T(), "BCDA_CA_FILE")

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: could not read CA file: read .: is a directory")
}

func (s *SSASClientTestSuite) TestNewSSASClient_NoURL() {
	conf.UnsetEnv(s.T(), "SSAS_USE_TLS")
	conf.UnsetEnv(s.T(), "SSAS_URL")

	client, err := authclient.NewSSASClient()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), client)
	assert.EqualError(s.T(), err, "SSAS client could not be created: no URL provided")
}

func (s *SSASClientTestSuite) TestNewSSASClient_TLSFalseNoURL() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.UnsetEnv(s.T(), "SSAS_URL")

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

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
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
		_, err := w.Write([]byte(`{"system_id": "1", "client_id":` + constants.FakeClientIDBt + `, "client_secret": ` + constants.FakeSecretBt + `, "client_name": "fake-name"}`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
	}

	resp, err := client.CreateSystem("fake-name", "fake-group", "fake-scope", "fake-key", "fake-tracking", nil)
	assert.Nil(s.T(), err)
	creds := auth.Credentials{}
	err = json.Unmarshal(resp, &creds)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), constants.FakeClientID, creds.ClientID)
	assert.Equal(s.T(), constants.FakeSecret, creds.ClientSecret)
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

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
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
		fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, constants.FakeClientID, constants.FakeSecret)
	})
	server := httptest.NewServer(router)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
	}

	resp, err := client.ResetCredentials("1")
	assert.Nil(s.T(), err)
	creds := auth.Credentials{}
	err = json.Unmarshal(resp, &creds)
	assert.Nil(s.T(), err, nil)
	assert.Equal(s.T(), constants.FakeClientID, creds.ClientID)
	assert.Equal(s.T(), constants.FakeSecret, creds.ClientSecret)
}

func (s *SSASClientTestSuite) TestDeleteCredentials() {
	router := chi.NewRouter()
	router.Delete("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
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

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
	}

	err = client.RevokeAccessToken("abc-123")
	assert.Nil(s.T(), err)
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

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
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

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
	}
	version, err := client.GetVersion()
	assert.Equal(s.T(), "", version)
	assert.NotNil(s.T(), err)
}

func (s *SSASClientTestSuite) TestCallSSASIntrospect() {
	server := testUtils.MakeTestServerWithIntrospectEndpoint(true)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "happy")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "customer")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.FailNow(constants.CreateSsasErr, err.Error())
	}

	const tokenString = "totallyfake.tokenstringfor.testing"

	bytes, err := client.CallSSASIntrospect(tokenString)
	if err != nil {
		s.FailNow("unexpected failure", err.Error())
	}

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), bytes)
	//assert.IsType(s.T(), &authclient.SSASClient{}, client)
}

func createByteIntrospectResponse(s *SSASClientTestSuite, shouldBeActive bool) (body []byte) {
	body, err := json.Marshal(struct {
		Active bool `json:"active"`
	}{Active: shouldBeActive})
	if err != nil {
		s.FailNow("Invalid response in mock ssas server")
	}
	return body
}

func (s *SSASClientTestSuite) TestCallSSASIntrospectEnvironmentVariables() {
	server := testUtils.MakeTestServerWithIntrospectEndpoint(true)
	emptyString := ""
	nonEmptyString := "happyCustomer"
	const token = "totallyfake.tokenstringfor.testing"

	tests := []struct {
		scenarioName                string
		bytesToReturn               []byte
		errTypeToReturn             error
		tokenString                 string
		envVariableSSASPublicURL    string
		envVariableSSASClientId     string
		envVariableSSASClientSecret string
	}{
		{"Active Valid Token", createByteIntrospectResponse(s, true), nil, token, server.URL, nonEmptyString, nonEmptyString},
		{"Empty Env variable - Public URL", nil, &customErrors.ConfigError{Msg: emptyString, Err: nil}, token, emptyString, nonEmptyString, nonEmptyString},
		{"Empty Env variable - Client Id", nil, &customErrors.ConfigError{Msg: emptyString, Err: nil}, token, server.URL, emptyString, nonEmptyString},
		{"Empty Env variable - Client Secret", nil, &customErrors.ConfigError{Msg: emptyString, Err: nil}, token, server.URL, nonEmptyString, emptyString},
	}

	for _, tt := range tests {
		s.T().Run(tt.scenarioName, func(t *testing.T) {

			conf.SetEnv(t, "SSAS_URL", server.URL)        //to start up appropriately
			conf.SetEnv(t, "SSAS_PUBLIC_URL", server.URL) //to start up appropriately
			conf.SetEnv(t, "SSAS_USE_TLS", "false")
			conf.SetEnv(t, "BCDA_SSAS_CLIENT_ID", tt.envVariableSSASClientId)
			conf.SetEnv(t, "BCDA_SSAS_SECRET", tt.envVariableSSASClientSecret)

			client, err := authclient.NewSSASClient()
			if err != nil {
				s.FailNow("Failed to create SSAS client", err.Error())
			}

			conf.SetEnv(t, "SSAS_URL", tt.envVariableSSASPublicURL)        //using test variable for gathering env variables
			conf.SetEnv(t, "SSAS_PUBLIC_URL", tt.envVariableSSASPublicURL) //using test variable for gathering env variables

			bytes, err := client.CallSSASIntrospect(tt.tokenString)
			assert.Equal(t, tt.bytesToReturn, bytes)
			assert.IsType(t, tt.errTypeToReturn, err)
		})
	}
}

func (s *SSASClientTestSuite) TestCallSSASIntrospectResponseHandling() {

	const token = "totallyfake.tokenstringfor.testing"
	emptyString := ""
	nonEmptyString := "123"
	fiveSeconds := "5"
	fiveHundredSeconds := "500"

	tests := []struct {
		scenarioName    string
		server          *httptest.Server
		sSasTimeout     string
		bytesToReturn   []byte
		errTypeToReturn error
		tokenString     string
	}{
		{"Active Valid Token", testUtils.MakeTestServerWithIntrospectEndpoint(true), fiveHundredSeconds, createByteIntrospectResponse(s, true), nil, token},
		{"Inactive (Expired) Valid Token", testUtils.MakeTestServerWithIntrospectEndpoint(false), fiveHundredSeconds, createByteIntrospectResponse(s, false), nil, token}, //no Expired Token error assertion here, that is handled in upstream method
		{"Introspect call timed out", testUtils.MakeTestServerWithIntrospectTimeout(), fiveSeconds, nil, &customErrors.RequestTimeoutError{Msg: emptyString, Err: nil}, token},
		{"Introspect call return other status code (502)", testUtils.MakeTestServerWithIntrospectReturn502(), fiveHundredSeconds, nil, &customErrors.UnexpectedSSASError{Msg: emptyString, Err: nil}, token},
	}

	for _, tt := range tests {
		s.T().Run(tt.scenarioName, func(t *testing.T) {

			conf.SetEnv(t, "SSAS_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_PUBLIC_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_USE_TLS", "false")
			conf.SetEnv(t, "BCDA_SSAS_CLIENT_ID", nonEmptyString)
			conf.SetEnv(t, "BCDA_SSAS_SECRET", nonEmptyString)
			conf.SetEnv(t, "SSAS_TIMEOUT_MS", tt.sSasTimeout)

			client, err := authclient.NewSSASClient()
			if err != nil {
				s.FailNow("Failed to create SSAS client", err.Error())
			}

			bytes, err := client.CallSSASIntrospect(tt.tokenString)
			assert.Equal(t, tt.bytesToReturn, bytes)
			assert.IsType(t, tt.errTypeToReturn, err)
		})
	}
}

func (s *SSASClientTestSuite) TestSSASClientTokenAuthentication() {
	const clientId, clientSecret, token = "happy", "client", "goodToken"

	tests := []struct {
		scenarioName    string
		server          *httptest.Server
		sSasTimeout     string
		bytesToReturn   []byte
		errTypeToReturn error
		expiresIn       []byte
	}{
		{"Active Credentials", testUtils.MakeTestServerWithValidTokenRequestEndpoint(), constants.FiveHundredSeconds, []byte(token), nil, []byte(constants.ExpiresInDefault)},
		{"Invalid Credentials", testUtils.MakeTestServerWithInvalidTokenRequestEndpoint(), constants.FiveHundredSeconds, []byte(nil), errors.New("Unauthorized"), []byte(nil)},
		{"Token request timed out", testUtils.MakeTestServerWithTokenRequestTimeout(), constants.FiveSeconds, []byte(nil), &customErrors.RequestTimeoutError{Msg: constants.EmptyString, Err: nil}, []byte(nil)},
	}

	for _, tt := range tests {
		s.T().Run(tt.scenarioName, func(t *testing.T) {
			conf.SetEnv(t, "SSAS_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_PUBLIC_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_TIMEOUT_MS", tt.sSasTimeout)

			client, err := authclient.NewSSASClient()
			if err != nil {
				log.Fatalf(constants.SsasClientErr, err.Error())
			}

			tokenString, expiresIn, err := client.GetToken(authclient.Credentials{ClientID: clientId, ClientSecret: clientSecret})

			assert.Equal(t, tt.bytesToReturn, tokenString)
			assert.Equal(t, tt.expiresIn, expiresIn)
			assert.IsType(t, tt.errTypeToReturn, err)
		})
	}
}

func TestSSASClientTestSuite(t *testing.T) {
	suite.Run(t, new(SSASClientTestSuite))
}
