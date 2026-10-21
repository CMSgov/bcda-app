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

// test all endpoints without token
// test all endpoints with token
// test shared export with DPC creds
// test shared export with non-DPC creds

// TODO
// need to be able to run this from makefile
// env vars different from executing from file vs make

var (
	clientID     string
	clientSecret string
	apiHost      string
	proto        string
	endpoint     string
	insecure     bool
)

type token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	Token_type  string `json:"token_type"`
}

type SmokeTests struct {
	suite.Suite
	ctx          context.Context
	host         string
	port         string
	token        token
	clientID     string
	clientSecret string
}

func TestSmokeTestsSuite(t *testing.T) {
	suite.Run(t, new(SmokeTests))
}

func (s *SmokeTests) SetupSuite() {
	db := database.Connect()
	p := auth.NewProvider(db)
	a, err := p.ResetSecret("0c527d2e-2e8a-4808-b11d-0fa06baf8254")
	require.NoError(s.T(), err)
	s.clientID = a.ClientID
	s.clientSecret = a.ClientSecret

}

func (s *SmokeTests) TearDownSuite() {}

func (s *SmokeTests) SetupTest() {}

func (s *SmokeTests) TearDownTest() {}

func (s *SmokeTests) TestPostToken() {

	t, err := s.PostToken()
	require.NoError(s.T(), err)
	require.Contains(s.T(), t.Token_type, "bearer")
	require.NotEmpty(s.T(), t.AccessToken)
	require.Equal(s.T(), t.ExpiresIn, "1200")

}

func (s *SmokeTests) TestPostSharedExport() {
	t, err := s.PostToken()
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), t)

	client := &http.Client{}

	body := strings.NewReader(`{"since": "2026-08-01T08:00:00-05:00", "version": "v3", "resource_types": ["Patient"], "partner_id": "test-shared-export", "data_types": ["adjudicated"], "mbis": []}`)

	req, _ := http.NewRequest("POST", "http://localhost:3000/api/v3/shared_export", body)

	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Authorization", "Bearer "+t.AccessToken)
	req.Header.Add("User-Agent", "bcda-api-e2e-test")

	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 202, resp.StatusCode)
	require.Contains(s.T(), resp.Header.Get("Content-Location"), "api/v3/jobs")

	defer resp.Body.Close()

}

// help func: get a token for A9994
func (s *SmokeTests) PostToken() (token, error) {

	client := &http.Client{}

	// url := fmt.Sprintf("%s://%s/token", proto, apiHost)

	var t = token{}

	// TODO run this for other envs
	req, _ := http.NewRequest("POST", "http://localhost:3000/auth/token", nil)
	req.SetBasicAuth(s.clientID, s.clientSecret)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "bcda-api-e2e-test")

	resp, err := client.Do(req)
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

// endpoint: /$SharedExport
func postSharedExport() string {
	// 	url := fmt.Sprintf("%s://%s/token", proto, apiHost)

	// 	// Prepare basic auth header
	// 	req, _ := http.NewRequest("POST", url, nil)
	// 	req.SetBasicAuth(clientID, clientSecret)
	// 	authHeader := req.Header.Get("Authorization")

	// 	header := map[string][]string{
	// 		"Accept":        {"application/json"},
	// 		"Authorization": {authHeader},
	// 		"User-Agent":    {"bcda-api-e2e-test"},
	// 	}
	// }

	// func makeIntrospectTarget(accessToken string) vegeta.Targeter {
	// 	url := fmt.Sprintf("%s://%s/introspect", proto, apiHost)

	// 	// Prepare basic auth header
	// 	req, _ := http.NewRequest("POST", url, nil)
	// 	req.SetBasicAuth(clientID, clientSecret)
	// 	authHeader := req.Header.Get("Authorization")

	// 	header := map[string][]string{
	// 		"Accept":        {"application/json"},
	// 		"Content-Type":  {"application/json"},
	// 		"Authorization": {authHeader},
	// 		"User-Agent":    {"bcda-ssas-performance-test"},
	// 	}

	// 	body := map[string]string{
	// 		"token": accessToken,
	// 	}
	// 	bodyBytes, _ := json.Marshal(body)

	//	return vegeta.NewStaticTargeter(vegeta.Target{
	//		Method: "POST",
	//		URL:    url,
	//		Header: header,
	//		Body:   bodyBytes,
	//	})
	return ""
}

// func runPerformanceTest(target vegeta.Targeter) (*plot.Plot, *vegeta.Metrics) {
// 	fmt.Printf("running performance test for: %s\n", endpoint)
// 	title := plot.Title(fmt.Sprintf("Performance Test - %s", endpoint))
// 	p := plot.New(title)
// 	defer p.Close()

// 	d := time.Second * time.Duration(duration)
// 	rate := vegeta.Rate{Freq: freq, Per: time.Second}
// 	var metrics vegeta.Metrics
// 	defer func() {
// 		// Needed to compute all of the summary metrics
// 		metrics.Close()
// 		if err := validateMetrics(metrics); err != nil {
// 			log.Printf("Validation error: %s", err.Error())
// 		}
// 	}()
// 	plotAttack(p, &metrics, target, rate, d)

// 	return p, &metrics
// }

// func plotAttack(p *plot.Plot, m *vegeta.Metrics, t vegeta.Targeter, r vegeta.Rate, du time.Duration) {
// 	attacker := vegeta.NewAttacker(
// 		vegeta.TLSConfig(&tls.Config{InsecureSkipVerify: insecure}), // #nosec G402
// 	)
// 	for results := range attacker.Attack(t, r, du, fmt.Sprintf("%dps:", r.Freq)) {
// 		if err := p.Add(results); err != nil {
// 			panic(err)
// 		}
// 		m.Add(results)
// 	}
// }

// func writeResults(filename string, data []byte) {
// 	re := regexp.MustCompile(`[^a-zA-Z0-9\.\-]`)
// 	clean := re.ReplaceAllString(filename, "-")
// 	if len(data) > 0 {
// 		fn := fmt.Sprintf("%s/%s", reportFilePath, clean)
// 		fmt.Printf("Writing results: %s\n", fn)
// 		err := os.WriteFile(fn, data, 0600)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// }

// func getAccessToken(cID, cSecret string) string {
// 	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/token", proto, apiHost), nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	req.SetBasicAuth(cID, cSecret)
// 	req.Header.Add("Accept", "application/json")

// 	fmt.Printf("Fetching test access token for introspect test...\n")
// 	tr := &http.Transport{
// 		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, // #nosec G402
// 	}
// 	client := &http.Client{Transport: tr}
// 	resp, err := client.Do(req) // #nosec G704
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to get token: %s", err.Error()))
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		panic(fmt.Sprintf("failed to get token: received status code %d", resp.StatusCode))
// 	}

// 	var t map[string]interface{}
// 	if err = json.NewDecoder(resp.Body).Decode(&t); err != nil {
// 		panic(fmt.Sprintf("unexpected token response format: %s", err.Error()))
// 	}

// 	tokenStr, ok := t["access_token"].(string)
// 	if !ok {
// 		panic("access_token missing or not a string in response")
// 	}
// 	return tokenStr
// }

// func validateMetrics(metrics vegeta.Metrics) error {
// 	if metrics.Requests == 0 {
// 		return nil
// 	}

// 	if len(metrics.Errors) > 0 {
// 		return fmt.Errorf("encountered %v errors", metrics.Errors)
// 	}

// 	if metrics.Success < 1.0 {
// 		return fmt.Errorf("expected success rate of 1.0, received %f",
// 			metrics.Success)
// 	}

// 	return nil
// }
