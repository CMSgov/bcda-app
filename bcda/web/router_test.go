package web

import (
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
	apiRouter  http.Handler
	dataRouter http.Handler
}

func (s *RouterTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	s.apiRouter = NewAPIRouter()
	s.dataRouter = NewDataRouter()
}

func (s *RouterTestSuite) getAPIRoute(route string) *http.Response {
	req := httptest.NewRequest("GET", route, nil)
	rr := httptest.NewRecorder()
	s.apiRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *RouterTestSuite) getDataRoute(route string) *http.Response {
	req := httptest.NewRequest("GET", route, nil)
	rr := httptest.NewRecorder()
	s.dataRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *RouterTestSuite) TestDefaultRoute() {
	res := s.getAPIRoute("/")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
	body, _ := ioutil.ReadAll(res.Body)
	assert.NotContains(s.T(), string(body), "404 page not found", "Default route returned wrong body")
	assert.Contains(s.T(), string(body), "Beneficiary Claims Data API")
}

func (s *RouterTestSuite) TestDataRoute() {
	res := s.getDataRoute("/data/test/test.ndjson")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *RouterTestSuite) TestFileServerRoute() {
	res := s.getAPIRoute("/api/v1/swagger")
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode)

	res = s.getAPIRoute("/api/v1/swagger/")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	r := chi.NewRouter()
	// Set up a bad route.  DON'T do this in real life
	assert.Panics(s.T(), func() {
		FileServer(r, "/api/v1/swagger{}", http.Dir("./swaggerui"))
	})
}

func (s *RouterTestSuite) TestMetadataRoute() {
	res := s.getAPIRoute("/api/v1/metadata")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	bytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.Nil(s.T(), err)
	assert.Contains(s.T(), string(bytes), `"resourceType":"CapabilityStatement"`)
}

func (s *RouterTestSuite) TestHealthRoute() {
	res := s.getAPIRoute("/_health")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestVersionRoute() {
	res := s.getAPIRoute("/_version")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestEOBExportRoute() {
	res := s.getAPIRoute("/api/v1/ExplanationOfBenefit/$export")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *RouterTestSuite) TestPatientExportRoute() {
	origPtExp := os.Getenv("ENABLE_PATIENT_EXPORT")
	defer os.Setenv("ENABLE_PATIENT_EXPORT", origPtExp)

	os.Setenv("ENABLE_PATIENT_EXPORT", "true")
	req := httptest.NewRequest("GET", "/api/v1/Patient/$export", nil)
	rr := httptest.NewRecorder()
	NewAPIRouter().ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	os.Setenv("ENABLE_PATIENT_EXPORT", "false")
	rr = httptest.NewRecorder()
	NewAPIRouter().ServeHTTP(rr, req)
	res = rr.Result()
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	os.Unsetenv("ENABLE_PATIENT_EXPORT")
	rr = httptest.NewRecorder()
	NewAPIRouter().ServeHTTP(rr, req)
	res = rr.Result()
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func (s *RouterTestSuite) TestCoverageExportRoute() {
	origCovExp := os.Getenv("ENABLE_COVERAGE_EXPORT")
	defer os.Setenv("ENABLE_COVERAGE_EXPORT", origCovExp)

	os.Setenv("ENABLE_COVERAGE_EXPORT", "true")
	req := httptest.NewRequest("GET", "/api/v1/Coverage/$export", nil)
	rr := httptest.NewRecorder()
	NewAPIRouter().ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	os.Setenv("ENABLE_COVERAGE_EXPORT", "false")
	rr = httptest.NewRecorder()
	NewAPIRouter().ServeHTTP(rr, req)
	res = rr.Result()
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	os.Unsetenv("ENABLE_COVERAGE_EXPORT")
	rr = httptest.NewRecorder()
	NewAPIRouter().ServeHTTP(rr, req)
	res = rr.Result()
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func (s *RouterTestSuite) TestJobStatusRoute() {
	res := s.getAPIRoute("/api/v1/jobs/1")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
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
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode, "http to https redirect return correct status code")
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
	assert.Equal(s.T(), http.StatusMethodNotAllowed, res.StatusCode, "http to https redirect rejects POST requests")
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
