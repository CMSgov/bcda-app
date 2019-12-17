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
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode)
}

func (s *RouterTestSuite) TestUGRoute() {
	res := s.getAPIRoute("/user_guide.html")
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode)
}

func (s *RouterTestSuite) TestDefaultProdRoute() {
	err := os.Setenv("DEPLOYMENT_TARGET", "prod")
	if err != nil {
		s.FailNow("err in setting env var", err)
	}
	// Need a new router because the one in the test setup does not use the environment variable set in this test.
	s.apiRouter = NewAPIRouter()
	res := s.getAPIRoute("/")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	err = os.Unsetenv("DEPLOYMENT_TARGET")
	if err != nil {
		s.FailNow("err in setting env var", err)
	}
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
	res := s.getAPIRoute("/api/v1/Patient/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Patients/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/all/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/all/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

}

func (s *RouterTestSuite) TestPatientExportRoute() {
	res := s.getAPIRoute("/api/v1/Patient/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Patients/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/all/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/all/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

}

func (s *RouterTestSuite) TestCoverageExportRoute() {
	res := s.getAPIRoute("/api/v1/Patient/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Patients/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/all/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/all/$export?_type=Coverage")
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
