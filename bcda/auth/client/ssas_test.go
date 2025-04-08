package client_test

import (
	"encoding/json"
	"fmt"
	"io"
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

type EnvVars struct {
	SSAS_URL            string
	SSAS_PUBLIC_URL     string
	METHOD              string
	BCDA_SSAS_CLIENT_ID string
}

func (s *SSASClientTestSuite) setEnvVars(r EnvVars) {
	if r.SSAS_URL != "" {
		if r.SSAS_URL == "-1" {
			conf.SetEnv(s.T(), "SSAS_URL", "")
		} else {
			conf.SetEnv(s.T(), "SSAS_URL", r.SSAS_URL)
		}
	}
	if r.SSAS_PUBLIC_URL != "" {
		if r.SSAS_PUBLIC_URL == "-1" {
			conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", "")
		} else {
			conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", r.SSAS_PUBLIC_URL)
		}
	}

	if r.BCDA_SSAS_CLIENT_ID != "" {
		if r.BCDA_SSAS_CLIENT_ID == "-1" {
			conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "")
		} else {
			conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", r.BCDA_SSAS_CLIENT_ID)
		}
	}
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

// Send along an invalid environment variable. Should still succeed.
func (s *SSASClientTestSuite) TestNewSSASClient_InvalidTimeout() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "SSAS_TIMEOUT_MS", "")

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

func (s *SSASClientTestSuite) TestNewSSASClient_NonFunctionalCert() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "true")
	conf.SetEnv(s.T(), "SSAS_URL", "https://ssas-url")
	conf.SetEnv(s.T(), "BCDA_CA_FILE", "../../../shared_files/bcda_invalid.ca.pem")

	_, err := authclient.NewSSASClient()
	assert.EqualError(s.T(), err, "SSAS client could not be created: could not append CA certificate(s)")
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

func (s *SSASClientTestSuite) TestCreateGroupTable() {
	tests := []struct {
		fnInput       []string
		header        int
		env           EnvVars
		errorExpected bool
		errorMessage  string
	}{
		{fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, header: http.StatusCreated, env: EnvVars{}, errorExpected: false, errorMessage: ""},
		{fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, header: http.StatusCreated, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, errorMessage: "Post \"localhost/group\": unsupported protocol scheme \"\""},
		{fnInput: []string{"\"", "name", "", `{ "ID": 123456 }`}, header: http.StatusCreated, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, errorMessage: "invalid input for new group_id \": parse \"\\n/group\": net/url: invalid control character in URL"},
		{fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, header: http.StatusCreated, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, errorMessage: "missing clientID or secret"},
		{fnInput: []string{"", "name", "", `{ "ID": 123456 }`}, env: EnvVars{}, header: http.StatusBadRequest, errorExpected: true, errorMessage: "{ \"ID\": 123456 }"},
	}

	for _, tc := range tests {
		router := chi.NewRouter()
		router.Post("/group", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.header)
			_, err := w.Write([]byte(tc.fnInput[3]))
			if err != nil {
				s.T().Fatal(err)
			}
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", server.URL)
		s.setEnvVars(tc.env)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}

		resp, err := client.CreateGroup(tc.fnInput[0], tc.fnInput[1], tc.fnInput[2])

		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.errorMessage)
		} else {
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), `{ "ID": 123456 }`, string(resp))
		}

	}
}
func (s *SSASClientTestSuite) TestDeleteGroupTable() {
	tests := []struct {
		fnInput       []string
		createGroup   bool
		ID            int
		header        int
		env           EnvVars
		errorExpected bool
		errorMessage  string
	}{
		{ID: 5, createGroup: false, fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, env: EnvVars{}, errorExpected: true, errorMessage: "could not delete group: "},
		{ID: 5, createGroup: false, fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, errorMessage: "could not delete group: Delete \"localhost/group/5\": unsupported protocol scheme \"\""},
		{ID: 5, createGroup: false, fnInput: []string{"\"", "name", "", `{ "ID": 123456 }`}, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, errorMessage: "could not delete group: parse \"\\n/group/5\": net/url: invalid control character in URL"},
		{ID: 5, createGroup: false, fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, errorMessage: "missing clientID or secret"},
		{ID: 123456, createGroup: true, fnInput: []string{"1", "name", "", `{ "ID": 123456 }`}, env: EnvVars{}, errorExpected: false},
	}

	for _, tc := range tests {
		router := chi.NewRouter()
		router.Post("/group", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := w.Write([]byte(tc.fnInput[3]))
			if err != nil {
				s.T().Fatal(err)
			}
		})
		router.Delete("/group/{id}", func(w http.ResponseWriter, r *http.Request) {
			if tc.createGroup == false || tc.ID != 123456 {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", server.URL)
		s.setEnvVars(tc.env)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}

		if tc.createGroup {
			_, err = client.CreateGroup(tc.fnInput[0], tc.fnInput[1], tc.fnInput[2])
			if err != nil {
				s.T().Fatal(err)
			}
		}
		err = client.DeleteGroup(tc.ID)
		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.errorMessage)
		} else {
			assert.Nil(s.T(), err)
		}

	}
}

func (s *SSASClientTestSuite) TestGetVersionTable() {
	tests := []struct {
		header        int
		env           EnvVars
		errorExpected bool
		message       string
		versionString string
	}{
		{header: http.StatusOK, env: EnvVars{}, errorExpected: false, message: "foo", versionString: "{\"version\":\"foo\"}\n"},
		{header: http.StatusOK, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, message: "parse \"\\n/_version\": net/url: invalid control character in URL", versionString: "{\"version\":\"foo\"}\n"},
		{header: http.StatusOK, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, message: "Get \"localhost/_version\": unsupported protocol scheme \"\"", versionString: "{\"version\":\"foo\"}\n"},
		{header: http.StatusOK, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret", versionString: "{\"version\":\"foo\"}\n"},
		{header: http.StatusOK, env: EnvVars{}, errorExpected: true, message: "Unable to parse version from response", versionString: "{\"foo\":\"foo\"}\n"},
		{header: http.StatusBadGateway, env: EnvVars{}, errorExpected: true, message: "SSAS server failed to return version ", versionString: "{\"version\":\"foo\"}\n"},
	}

	for _, tc := range tests {
		router := chi.NewRouter()
		router.Get("/_version", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tc.header)

			_, err := io.WriteString(w, tc.versionString)
			if err != nil {
				s.FailNow("Failed to create SSAS server", err.Error())
			}
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
		conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", server.URL)
		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()

		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}
		version, err := client.GetVersion()

		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), tc.message, version)
		}

	}
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
	assert.IsType(s.T(), &authclient.SSASClient{}, client)
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

func (s *SSASClientTestSuite) TestGetToken() {
	const clientId, clientSecret, token = "happy", "client", "goodToken"

	tests := []struct {
		scenarioName      string
		server            *httptest.Server
		sSasTimeout       string
		bytesToReturn     []byte
		errTypeToReturn   error
		expiresIn         []byte
		setInvalidSSASUrl bool
	}{
		{"Active Credentials", testUtils.MakeTestServerWithValidTokenRequest(), constants.FiveHundredSeconds, []byte(token), nil, []byte(constants.ExpiresInDefault), false},
		{"Invalid Carriage Character", testUtils.MakeTestServerWithInvalidCarriage(), constants.FiveHundredSeconds, []byte(nil), &customErrors.UnexpectedSSASError{Msg: constants.EmptyString, Err: nil}, []byte(nil), false},
		{"Invalid SSAS Url Structure", testUtils.MakeTestServerWithInvalidCarriage(), constants.FiveHundredSeconds, []byte(nil), &customErrors.InternalParsingError{Msg: constants.EmptyString, Err: nil}, []byte(nil), true},
		{"Invalid Request", testUtils.MakeTestServerWithBadRequest(), constants.FiveHundredSeconds, []byte(nil), &customErrors.SSASErrorBadRequest{Msg: constants.EmptyString, Err: nil}, []byte(nil), false},
		{"Token Request Timed Out", testUtils.MakeTestServerWithTokenRequestTimeout(), constants.FiveSeconds, []byte(nil), &customErrors.RequestTimeoutError{Msg: constants.EmptyString, Err: nil}, []byte(nil), false},
		{"Invalid Credentials", testUtils.MakeTestServerWithInvalidTokenRequest(), constants.FiveHundredSeconds, []byte(nil), &customErrors.SSASErrorUnauthorized{Msg: constants.EmptyString, Err: nil}, []byte(nil), false},
	}

	for _, tt := range tests {
		s.T().Run(tt.scenarioName, func(t *testing.T) {
			conf.SetEnv(t, "SSAS_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_PUBLIC_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_TIMEOUT_MS", tt.sSasTimeout)
			if tt.setInvalidSSASUrl {
				conf.SetEnv(t, "SSAS_PUBLIC_URL", "\n")
			}
			client, err := authclient.NewSSASClient()
			if err != nil {
				s.T().Fatal(err)
			}

			r := testUtils.ContextTransactionID()
			tokenInfo, err := client.GetToken(authclient.Credentials{ClientID: clientId, ClientSecret: clientSecret}, *r)

			assert.Contains(t, tokenInfo, string(tt.bytesToReturn))
			assert.Contains(t, tokenInfo, string(tt.expiresIn))
			assert.IsType(t, tt.errTypeToReturn, err)
		})
	}
}

func (s *SSASClientTestSuite) TestGetTokenHeaders() {
	const clientId, clientSecret, token = "happy", "client", "goodToken"
	router := chi.NewRouter()
	router.Post(constants.TokenPath, func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "token_type": "bearer", "access_token": "goodToken", "expires_in": "1200" }`))
		if err != nil {
			s.T().Fatal(err)
		}
		if r.Header.Get("transaction-id") == "" {
			s.T().Errorf("Expected transaction-id header value, got empty string")
		}
	})
	server := httptest.NewServer(router)
	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_TIMEOUT_MS", "500")

	client, err := authclient.NewSSASClient()
	if err != nil {
		s.T().Fatal(err)
	}

	r := testUtils.ContextTransactionID()
	_, err = client.GetToken(authclient.Credentials{ClientID: clientId, ClientSecret: clientSecret}, *r)

	assert.IsType(s.T(), nil, err)

}

func TestSSASClientTestSuite(t *testing.T) {
	suite.Run(t, new(SSASClientTestSuite))
}

func (s *SSASClientTestSuite) TestPingTable() {
	tests := []struct {
		fnInput       []string
		env           EnvVars
		errorExpected bool
		message       string
	}{
		{fnInput: []string{}, env: EnvVars{}, errorExpected: false},
		{fnInput: []string{}, env: EnvVars{BCDA_SSAS_CLIENT_ID: "fake"}, errorExpected: true, message: "introspect request failed; 401"},
		{fnInput: []string{}, env: EnvVars{SSAS_PUBLIC_URL: "\n"}, errorExpected: true, message: "bad request structure: parse \"\\n/introspect\": net/url: invalid control character in URL"},
		{fnInput: []string{}, env: EnvVars{SSAS_PUBLIC_URL: "localhost"}, errorExpected: true, message: "introspect request failed: Post \"localhost/introspect\": unsupported protocol scheme \"\""},
		{fnInput: []string{}, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret"},
	}

	for _, tc := range tests {

		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()

		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}
		err = client.Ping()

		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			assert.Nil(s.T(), err)
		}

	}
}

func (s *SSASClientTestSuite) TestGetPublicKeyTable() {
	tests := []struct {
		fnInput       []string
		header        int
		env           EnvVars
		errorExpected bool
		message       string
	}{
		{fnInput: []string{}, header: http.StatusOK, env: EnvVars{}, errorExpected: false, message: ""},
		{fnInput: []string{}, header: http.StatusOK, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, message: "could not get public key: parse \"\\n/system/1/key\": net/url: invalid control character in URL"},
		{fnInput: []string{}, header: http.StatusOK, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, message: "could not get public key: Get \"localhost/system/1/key\": unsupported protocol scheme \"\""},
		{fnInput: []string{}, header: http.StatusOK, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret"},
	}

	for _, tc := range tests {
		router := chi.NewRouter()
		keyStr := "123456"
		router.Get("/system/{systemID}/key", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte(`{ "client_id": "123456", "public_key": "` + keyStr + `" }`))
			if err != nil {
				s.T().Fatal(err)
			}
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}

		respKey, err := client.GetPublicKey(1)

		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), keyStr, string(respKey))
		}

	}
}
func (s *SSASClientTestSuite) TestResetCredentialsTable() {
	tests := []struct {
		fnInput       []string
		header        int
		env           EnvVars
		errorExpected bool
		message       string
	}{
		{fnInput: []string{}, header: http.StatusCreated, env: EnvVars{}, errorExpected: false, message: ""},
		{fnInput: []string{}, header: http.StatusBadGateway, env: EnvVars{}, errorExpected: true, message: "failed to reset credentials. status code: 502"},
		{fnInput: []string{}, header: http.StatusCreated, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, message: "failed to reset credentials: parse \"\\n/system/1/credentials\": net/url: invalid control character in URL"},
		{fnInput: []string{}, header: http.StatusCreated, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, message: "failed to reset credentials: Put \"localhost/system/1/credentials\": unsupported protocol scheme \"\""},
		{fnInput: []string{}, header: http.StatusCreated, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret"},
	}

	for _, tc := range tests {

		router := chi.NewRouter()
		router.Put("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.header)
			fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, constants.FakeClientID, constants.FakeSecret)
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}
		resp, err := client.ResetCredentials("1")
		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			assert.Nil(s.T(), err, nil)
			creds := auth.Credentials{}
			err = json.Unmarshal(resp, &creds)
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), constants.FakeClientID, creds.ClientID)
			assert.Equal(s.T(), constants.FakeSecret, creds.ClientSecret)
		}

	}
}

func (s *SSASClientTestSuite) TestDeleteCredentialsTable() {
	tests := []struct {
		fnInput       []string
		header        int
		env           EnvVars
		errorExpected bool
		message       string
	}{
		{fnInput: []string{}, header: http.StatusOK, env: EnvVars{}, errorExpected: false, message: ""},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{}, errorExpected: true, message: "failed to delete credentials; 404"},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, message: "failed to delete credentials: parse \"\\n/system/1/credentials\": net/url: invalid control character in URL"},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, message: "failed to delete credentials: Delete \"localhost/system/1/credentials\": unsupported protocol scheme \"\""},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret"},
	}

	for _, tc := range tests {

		router := chi.NewRouter()
		router.Delete("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.header)
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}
		err = client.DeleteCredentials("1")
		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			assert.Nil(s.T(), err, nil)
		}

	}
}
func (s *SSASClientTestSuite) TestRevokeAccessTokenTable() {
	tests := []struct {
		fnInput       []string
		header        int
		env           EnvVars
		errorExpected bool
		message       string
	}{
		{fnInput: []string{}, header: http.StatusOK, env: EnvVars{}, errorExpected: false, message: ""},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{}, errorExpected: true, message: "failed to revoke token; 404"},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, message: "bad request structure: parse \"\\n/token/abc-123\": net/url: invalid control character in URL"},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, message: "failed to revoke token: Delete \"localhost/token/abc-123\": unsupported protocol scheme \"\""},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret"},
	}

	for _, tc := range tests {

		router := chi.NewRouter()
		router.Delete("/token/{tokenID}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.header)
		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}
		err = client.RevokeAccessToken("abc-123")
		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			assert.Nil(s.T(), err, nil)
		}

	}
}

func (s *SSASClientTestSuite) TestCreateSystemTable() {
	tests := []struct {
		fnInput       []string
		header        int
		env           EnvVars
		errorExpected bool
		message       string
	}{
		{fnInput: []string{}, header: http.StatusCreated, env: EnvVars{}, errorExpected: false, message: ""},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{}, errorExpected: true, message: "failed to create system. status code: 404"},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{SSAS_URL: "\n"}, errorExpected: true, message: "failed to create system: parse \"\\n/system\": net/url: invalid control character in URL"},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{SSAS_URL: "localhost"}, errorExpected: true, message: "failed to create system: Post \"localhost/system\": unsupported protocol scheme \"\""},
		{fnInput: []string{}, header: http.StatusNotFound, env: EnvVars{BCDA_SSAS_CLIENT_ID: "-1"}, errorExpected: true, message: "missing clientID or secret"},
	}

	for _, tc := range tests {

		router := chi.NewRouter()
		router.Post("/system", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.header)
			if tc.header == http.StatusCreated {
				_, err := w.Write([]byte(`{"system_id": "1", "client_id":` + constants.FakeClientIDBt + `, "client_secret": ` + constants.FakeSecretBt + `, "client_name": "fake-name"}`))
				if err != nil {
					s.T().Fatal(err)
				}
			}

		})
		server := httptest.NewServer(router)

		conf.SetEnv(s.T(), "SSAS_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
		conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

		s.setEnvVars(tc.env)
		client, err := authclient.NewSSASClient()
		if err != nil {
			s.FailNow(constants.CreateSsasErr, err.Error())
		}
		resp, err := client.CreateSystem("fake-name", "fake-group", "fake-scope", "fake-key", "fake-tracking", nil)

		if tc.errorExpected {
			assert.EqualError(s.T(), err, tc.message)
		} else {
			creds := auth.Credentials{}
			err = json.Unmarshal(resp, &creds)
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), constants.FakeClientID, creds.ClientID)
			assert.Equal(s.T(), constants.FakeSecret, creds.ClientSecret)
		}

	}
}
