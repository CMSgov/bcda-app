package smoke

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TODO: BCDA-10508
// - need to be able to run this from makefile
// - local configs (env vars, docker env vars, vs test settings, multiple local dbs) need
// to be resolved to add this.
// - these run with vscode go test vars (database_url needs to point to containers) and if local.env has the endpoint enabled

type clientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	Token_type  string `json:"token_type"`
}

type SharedBackendSmokeTests struct {
	suite.Suite
	host               string
	sharedAccessACO    clientCredentials
	nonSharedAccessACO clientCredentials
	environment        string
}

// TODO: BCDA-10508 temporary only run in dev and test
func TestSmokeTestsSuite(t *testing.T) {
	t.Skip()
	environment := conf.GetEnv("DEPLOYMENT_TARGET")
	if environment == "dev" || environment == "test" {
		suite.Run(t, new(SharedBackendSmokeTests))
	}
}

func (s *SharedBackendSmokeTests) SetupSuite() {
	// TODO: BCDA-10508 configure to run locally
	// if conf.GetEnv("ENV") == "local" {
	// 	s.configureLocalTest()
	// 	return
	// }

	sea := conf.GetEnv("SHARED_EXPORT_ACO")
	require.NotEmpty(s.T(), sea, "SHARED_EXPORT_ACO env var not set")
	err := json.Unmarshal([]byte(sea), &s.sharedAccessACO)
	require.NoError(s.T(), err, "failed to unmarshal ACO credentials")

	nsea := conf.GetEnv("NON_SHARED_EXPORT_ACO")
	require.NotEmpty(s.T(), nsea, "NON_SHARED_EXPORT_ACO env var not set")
	err = json.Unmarshal([]byte(nsea), &s.nonSharedAccessACO)
	require.NoError(s.T(), err)

	s.host = conf.GetEnv("API_HOST")
	require.NotEmpty(s.T(), s.host, "API_HOST env var not set")
}

func (s *SharedBackendSmokeTests) TearDownSuite() {}

func (s *SharedBackendSmokeTests) SetupTest() {}

func (s *SharedBackendSmokeTests) TearDownTest() {}

func (s *SharedBackendSmokeTests) TestPostToken() {

	t, err := s.postToken(s.sharedAccessACO.ClientID, s.sharedAccessACO.ClientSecret)
	require.NoError(s.T(), err)
	require.Contains(s.T(), t.Token_type, "bearer")
	require.NotEmpty(s.T(), t.AccessToken)
	require.Equal(s.T(), t.ExpiresIn, "1200")

}

// TODO: (BCDA-10498) remove version checks and add validation for data and resources types
// get synth MBIs from v3 test aco in each environment
func (s *SharedBackendSmokeTests) TestPostSharedExport_SharedPartner() {

	tests := []struct {
		name       string
		body       *strings.Reader
		respStatus int
	}{
		{"Start shared export - valid body", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "test-shared-export", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusAccepted},
		{"Start shared export - empty resource type", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": [], "partner_id": "shared-export-partner", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusBadRequest},
		{"Start shared export - empty data type", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "shared-export-partner", "data_types": [], "mbis": ["1234"]}`), http.StatusBadRequest},
		{"Start shared export - empty mbi list", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "shared-export-partner", "data_types": ["adjudicated"], "mbis": []}`), http.StatusBadRequest},
	}
	t, err := s.postToken(s.sharedAccessACO.ClientID, s.sharedAccessACO.ClientSecret)
	require.NoError(s.T(), err)
	for _, tt := range tests {
		s.Run(tt.name, func() {
			require.NotEmpty(s.T(), t)
			j, err := s.postSharedExport(t.AccessToken, tt.body)
			require.NoError(s.T(), err)
			require.Equal(s.T(), tt.respStatus, j.StatusCode)
			if tt.respStatus == 202 {
				require.Contains(s.T(), j.Header.Get("Content-Location"), "api/v3/jobs")

			}
		})
	}

}

func (s *SharedBackendSmokeTests) TestPostSharedExport_NotSharedPartner() {

	t, err := s.postToken(s.nonSharedAccessACO.ClientID, s.nonSharedAccessACO.ClientSecret)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), t.AccessToken)

	body := strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "foo", "data_types": ["adjudicated"], "mbis": ["1234"]}`)
	j, err := s.postSharedExport(t.AccessToken, body)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 401, j.StatusCode, "non shared export partner should not be authorized to use /$shared_export")

}

// helper function
func (s *SharedBackendSmokeTests) postToken(client string, secret string) (token, error) {
	httpClient := &http.Client{}
	var t = token{}
	req, _ := http.NewRequest("POST", s.host+"/auth/token", nil)
	req.SetBasicAuth(client, secret)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "bcda-api-smoke-test")

	resp, err := httpClient.Do(req)
	if err != nil {
		return t, err
	}

	if resp.StatusCode != 200 {
		return t, fmt.Errorf("failed to get token: expected HTTP 200 but got: %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&t)
	if err != nil {
		return t, err
	}

	return t, err
}

// helper function: POST $SharedExport
// TODO: (BCDA-10498) parameterize request for local and other envs
func (s *SharedBackendSmokeTests) postSharedExport(token string, body *strings.Reader) (*http.Response, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("POST", s.host+"/api/v3/$shared_export", body)
	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("User-Agent", "bcda-api-smoke-test")

	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	defer resp.Body.Close()
	return resp, nil

}
