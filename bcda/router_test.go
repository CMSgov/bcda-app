package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	apiServer  *httptest.Server
	dataServer *httptest.Server
	rr         *httptest.ResponseRecorder
}

func (s *RouterTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	s.apiServer = httptest.NewServer(NewAPIRouter())
	s.dataServer = httptest.NewServer(NewDataRouter())
}

func (s *RouterTestSuite) TearDownTest() {
	s.apiServer.Close()
	s.dataServer.Close()
}

func (s *RouterTestSuite) TestDefaultRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, res.StatusCode)

	bytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.Nil(s.T(), err)
	r := string(bytes)
	assert.NotContains(s.T(), r, "404 page not found", "Default route returned wrong body")
	assert.Contains(s.T(), r, "Beneficiary Claims Data API")
}

func (s *RouterTestSuite) TestDataRoute() {
	res, err := s.apiServer.Client().Get(s.dataServer.URL + "/data/test/test.ndjson")
	assert.Nil(s.T(), err, fmt.Sprintf("error when getting data route: %s", err))
	assert.Equal(s.T(), 401, res.StatusCode)
}

func (s *RouterTestSuite) TestAuthTokenRoute() {
	res, err := s.apiServer.Client().Post(s.apiServer.URL+"/auth/token", "", nil)
	assert.Nil(s.T(), err, fmt.Sprintf("error getting auth token route: %s", err))
	assert.Equal(s.T(), 400, res.StatusCode)
}

func (s *RouterTestSuite) TestFileServerRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL + "/api/v1/swagger")
	assert.Nil(s.T(), err, fmt.Sprintf("error when getting swagger route: %s", err))
	assert.Equal(s.T(), 200, res.StatusCode)

	r := chi.NewRouter()
	// Set up a bad route.  DON'T do this in real life
	assert.Panics(s.T(), func() {
		FileServer(r, "/api/v1/swagger{}", http.Dir("./swaggerui"))
	})
}

func (s *RouterTestSuite) TestMetadataRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL + "/api/v1/metadata")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, res.StatusCode)

	bytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), string(bytes), "resourceType")
}

func (s *RouterTestSuite) TestTokenRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL + "/api/v1/token")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, res.StatusCode)
}

func (s *RouterTestSuite) TestHealthRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL + "/_health")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, res.StatusCode)
}

func (s *RouterTestSuite) TestVersionRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL + "/_version")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, res.StatusCode)
}

func (s *RouterTestSuite) TestEOBExportRoute() {
	res, err := s.apiServer.Client().Get(s.apiServer.URL + "/api/v1/ExplanationOfBenefit/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 401, res.StatusCode)
}

func (s *RouterTestSuite) TestPatientExportRoute() {
	origPtExp := os.Getenv("ENABLE_PATIENT_EXPORT")
	defer os.Setenv("ENABLE_PATIENT_EXPORT", origPtExp)

	os.Setenv("ENABLE_PATIENT_EXPORT", "true")
	apiServer := httptest.NewServer(NewAPIRouter())
	client := apiServer.Client()
	res, err := client.Get(apiServer.URL + "/api/v1/Patient/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 401, res.StatusCode)

	os.Setenv("ENABLE_PATIENT_EXPORT", "false")
	apiServer.Config.Handler = NewAPIRouter()
	res, err = client.Get(apiServer.URL + "/api/v1/Patient/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 404, res.StatusCode)

	os.Unsetenv("ENABLE_PATIENT_EXPORT")
	apiServer.Config.Handler = NewAPIRouter()
	res, err = client.Get(apiServer.URL + "/api/v1/Patient/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 404, res.StatusCode)

	apiServer.Close()
}

func (s *RouterTestSuite) TestCoverageExportRoute() {
	origCovExp := os.Getenv("ENABLE_COVERAGE_EXPORT")
	defer os.Setenv("ENABLE_COVERAGE_EXPORT", origCovExp)

	os.Setenv("ENABLE_COVERAGE_EXPORT", "true")
	apiServer := httptest.NewServer(NewAPIRouter())
	client := apiServer.Client()
	res, err := client.Get(apiServer.URL + "/api/v1/Coverage/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 401, res.StatusCode)

	os.Setenv("ENABLE_COVERAGE_EXPORT", "false")
	apiServer.Config.Handler = NewAPIRouter()
	res, err = client.Get(apiServer.URL + "/api/v1/Coverage/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 404, res.StatusCode)

	os.Unsetenv("ENABLE_COVERAGE_EXPORT")
	apiServer.Config.Handler = NewAPIRouter()
	res, err = client.Get(apiServer.URL + "/api/v1/Coverage/$export")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 404, res.StatusCode)

	apiServer.Close()
}

func (s *RouterTestSuite) TestJobStatusRoute() {

}

func (s *RouterTestSuite) TestHTTPServerRedirect() {
	router := NewHTTPRouter()

	// Redirect GET http requests to https
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()

	assert.Nil(s.T(), err, "redirect GET http to https")
	assert.Equal(s.T(), 301, res.StatusCode, "http to https redirect return correct status code")
	assert.Equal(s.T(), "close", res.Header.Get("Connection"), "http to https redirect sets 'connection: close' header")
	assert.Contains(s.T(), res.Header.Get("Location"), "https://", "location response header contains 'https://'")

	// Only respond to GET requests
	req, err = http.NewRequest("POST", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res = w.Result()

	assert.Nil(s.T(), err, "redirect POST http to https")
	assert.Equal(s.T(), 405, res.StatusCode, "http to https redirect rejects POST requests")
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
