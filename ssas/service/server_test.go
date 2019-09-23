package service

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const unitSigningKeyPath string = "../../shared_files/ssas/unit_test_private_key.pem"

type ServerTestSuite struct {
	suite.Suite
	server *Server
	info map[string][]string
}

func (s *ServerTestSuite) SetupSuite() {
	s.info = make(map[string][]string)
	s.info["public"] = []string{"token", "register"}
}

func (s *ServerTestSuite) SetupTest() {
	s.server = NewServer("test-server", ":9999", "9.99.999", s.info, nil, true, unitSigningKeyPath, 37 * time.Minute)
}

func (s *ServerTestSuite) TestNewServer() {
	assert.NotNil(s.T(), s.server)
	assert.NotNil(s.T(), s.server.tokenSigningKey)
	assert.NotEmpty(s.T(), s.server.name)
	assert.NotEmpty(s.T(), s.server.port)
	assert.NotEmpty(s.T(), s.server.version)
	assert.NotEmpty(s.T(), s.server.info)
	assert.NotEmpty(s.T(), s.server.router)
	assert.True(s.T(), s.server.notSecure)
	assert.NotNil(s.T(), s.server.tokenSigningKey)
	assert.NotZero(s.T(), s.server.tokenTTL)

	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
	})
	ts := NewServer("test-server", ":9999", "9.99.999", s.info, r, true, unitSigningKeyPath, 37 * time.Minute)
	assert.NotEmpty(s.T(), ts.router)
	routes, err := ts.ListRoutes()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), routes)
	expected := []string{"GET /_health", "GET /_info", "GET /_version", "GET /*/test"}
	assert.Equal(s.T(), expected, routes)
}

// test Server() ? how????

func (s *ServerTestSuite) TestGetInfo() {
	req := httptest.NewRequest("GET", "/_info", nil)
	handler := http.HandlerFunc(s.server.getInfo)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Result().StatusCode)
	b, _ := ioutil.ReadAll(rr.Result().Body)
	assert.Contains(s.T(), string(b), `{"public":["token","register"]}`)
}

func (s *ServerTestSuite) TestGetVersion() {
	req := httptest.NewRequest("GET", "/_version", nil)
	handler := http.HandlerFunc(s.server.getVersion)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Result().StatusCode)
	b, _ := ioutil.ReadAll(rr.Result().Body)
	assert.Contains(s.T(), string(b), "9.99.999")
}

func (s *ServerTestSuite) TestGetHealthCheck() {
	req := httptest.NewRequest("GET", "/_health", nil)
	handler := http.HandlerFunc(s.server.getHealthCheck)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Result().StatusCode)
	b, _ := ioutil.ReadAll(rr.Result().Body)
	assert.Contains(s.T(), string(b), `{"database":"ok"}`)
}

func (s *ServerTestSuite) TestNYI() {
	req := httptest.NewRequest("GET", "/random_endpoint", nil)
	handler := http.HandlerFunc(NYI)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Result().StatusCode)
	b, _ := ioutil.ReadAll(rr.Result().Body)
	assert.Contains(s.T(), string(b), "Not Yet Implemented")
}

// test ConnectionClose()

// MintToken(), MintTokenWithDuration()

func (s *ServerTestSuite) TestNewServerWithBadSigningKey() {
	ts := NewServer("test-server", ":9999", "9.99.999", s.info, nil, true, "", 37 * time.Minute)
	assert.Nil(s.T(), ts)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
