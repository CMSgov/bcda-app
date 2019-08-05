package service

import (
	"net/http"
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
		w.Write([]byte("test"))
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

// test getInfo(), getVersion(), getHealthCheck()

// test NYI()

// test ConnectionClose()

// MintToken(), MintTokenWithDuration(), mintToken()

func (s *ServerTestSuite) TestNewServerWithBadSigningKey() {
	ts := NewServer("test-server", ":9999", "9.99.999", s.info, nil, true, "", 37 * time.Minute)
	assert.Nil(s.T(), ts)
}

// TODO belongs in public and admin server testing
/*
func (s *ServerTestSuite) TestTokenDurationOverride() {
	originalValue := os.Getenv("SSAS_TOKEN_TTL_IN_MINUTES")
	assert.NotEmpty(s.T(), s.server.tokenTTL)
	assert.Equal(s.T(), time.Hour, s.server.tokenTTL)
	os.Setenv("SSAS_TOKEN_TTL_IN_MINUTES", "5")
	s.server.initTokenDuration()
	assert.Equal(s.T(), 5*time.Minute, s.server.tokenTTL)
	os.Setenv("SSAS_TOKEN_TTL_IN_MINUTES", originalValue)
}

func (s *ServerTestSuite) TestTokenDurationEmptyOverride() {
	assert.NotEmpty(s.T(), s.server.tokenTTL)
	assert.Equal(s.T(), time.Hour, s.server.tokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "")
	s.server.initTokenDuration()
	assert.Equal(s.T(), time.Hour, s.server.tokenTTL)
}
*/

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
