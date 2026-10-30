package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TODO: (BCDA-10498)
// need to be able to run this from makefile
// env vars different from executing from file vs make

type token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	Token_type  string `json:"token_type"`
}

type SharedBackendSmokeTests struct {
	suite.Suite
	ctx context.Context
	// host                  string
	// port                  string
	// token                 token
	dpcClientID           string
	dpcClientSecret       string
	nonSharedClientID     string
	nonSharedClientSecret string
}

func TestSmokeTestsSuite(t *testing.T) {
	suite.Run(t, new(SharedBackendSmokeTests))
}

func (s *SharedBackendSmokeTests) SetupSuite() {
	db := database.Connect()
	p := auth.NewProvider(db)
	dpc, err := p.ResetSecret("1db778aa-45eb-428e-99e4-d32c519f9d69")
	require.NoError(s.T(), err)
	nonShared, err := p.ResetSecret("0c527d2e-2e8a-4808-b11d-0fa06baf8254")
	require.NoError(s.T(), err)
	s.dpcClientID = dpc.ClientID
	s.dpcClientSecret = dpc.ClientSecret
	s.nonSharedClientID = nonShared.ClientID
	s.nonSharedClientSecret = nonShared.ClientSecret

}

func (s *SharedBackendSmokeTests) TearDownSuite() {}

func (s *SharedBackendSmokeTests) SetupTest() {}

func (s *SharedBackendSmokeTests) TearDownTest() {}

func (s *SharedBackendSmokeTests) TestPostToken() {

	t, err := s.PostToken(s.dpcClientID, s.dpcClientSecret)
	require.NoError(s.T(), err)
	require.Contains(s.T(), t.Token_type, "bearer")
	require.NotEmpty(s.T(), t.AccessToken)
	require.Equal(s.T(), t.ExpiresIn, "1200")

}

// TODO: (BCDA-10498) remove version checks and add validation for data and resources types
func (s *SharedBackendSmokeTests) TestPostSharedExport_SharedPartner() {

	tests := []struct {
		name       string
		body       *strings.Reader
		respStatus int
	}{
		{"DPC start shared export - valid body", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "DPC-1", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusAccepted},
		// DPC will only have access to v3 and it's validated in the request url
		// {"DPC start shared export - invalid version", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "153", "resource_types": ["Patient"], "partner_id": "DPC-1", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusBadRequest},
		// {"DPC start shared export - empty version", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "", "resource_types": ["Patient"], "partner_id": "DPC-1", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusBadRequest},
		// {"DPC start shared export - invalid resource type", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["foo"], "partner_id": "DPC-1", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusBadRequest},
		{"DPC start shared export - empty resource type", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": [], "partner_id": "DPC-1", "data_types": ["adjudicated"], "mbis": ["1234"]}`), http.StatusBadRequest},
		// {"DPC start shared export - invalid data type", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "DPC-1", "data_types": ["foo"], "mbis": ["1234"]}`), http.StatusBadRequest},
		{"DPC start shared export - empty data type", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "DPC-1", "data_types": [], "mbis": ["1234"]}`), http.StatusBadRequest},
		{"DPC start shared export - empty mbi list", strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "DPC-1", "data_types": ["adjudicated"], "mbis": []}`), http.StatusBadRequest},
	}
	t, err := s.PostToken(s.dpcClientID, s.dpcClientSecret)
	for _, tt := range tests {
		s.Run(tt.name, func() {
			//t, err := s.PostToken(s.dpcClientID, s.dpcClientSecret)
			require.NoError(s.T(), err)
			require.NotEmpty(s.T(), t)
			j, err := s.postSharedExport(t.AccessToken, tt.body)

			require.NoError(s.T(), err)
			require.NoError(s.T(), err)
			require.Equal(s.T(), tt.respStatus, j.StatusCode)
			if tt.respStatus == 202 {
				require.Contains(s.T(), j.Header.Get("Content-Location"), "api/v3/jobs")

			}
		})
	}
}

func (s *SharedBackendSmokeTests) TestPostSharedExport_NotSharedPartner() {
	t, err := s.PostToken(s.nonSharedClientID, s.nonSharedClientSecret)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), t)

	body := strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "foo", "data_types": ["adjudicated"], "mbis": ["1234"]}`)
	j, err := s.postSharedExport(t.AccessToken, body)

	require.NoError(s.T(), err)
	require.Equal(s.T(), 401, j.StatusCode)

}

// helper function
// TODO: (BCDA-10498) parameterize for local and other envs
func (s *SharedBackendSmokeTests) PostToken(client string, secret string) (token, error) {

	httpClient := &http.Client{}
	var t = token{}
	// url := fmt.Sprintf("%s://%s/token", proto, apiHost)

	req, _ := http.NewRequest("POST", "http://localhost:3000/auth/token", nil)
	req.SetBasicAuth(client, secret)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "bcda-api-e2e-test")

	resp, err := httpClient.Do(req)
	if err != nil {
		return t, err
	}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&t)
	if err != nil {
		return t, err
	}

	if resp.StatusCode != 200 {
		return t, fmt.Errorf("failed to get token: expected HTTP 200 but got: %d", resp.StatusCode)
	}

	return t, err
}

// helper function: POST $SharedExport
// TODO: (BCDA-10498) parameterize request for local and other envs
func (s *SharedBackendSmokeTests) postSharedExport(token string, body *strings.Reader) (*http.Response, error) {
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "http://localhost:3000/api/v3/shared_export", body)

	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("User-Agent", "bcda-api-e2e-test")

	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	return resp, nil

}
