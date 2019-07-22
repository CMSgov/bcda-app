package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const unitSigningKeyPath string = "../../shared_files/ssas/unit_test_private_key.pem"

type ServerTestSuite struct {
	suite.Suite
	server *Server
}

func (s *ServerTestSuite) SetupSuite() {
	info := make(map[string][]string)
	info["public"] = []string{"token", "register"}
	s.server = NewServer("test-server", ":9999", "9.99.999", info, s.server.newBaseRouter(), true)
}

func (s *ServerTestSuite) TearDownSuite() {
}

func (s *ServerTestSuite) TestSetSigningKeys() {
	err := s.server.SetSigningKeys("")
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "bad signing key", "testing bad key")

	err = s.server.SetSigningKeys(unitSigningKeyPath)
	assert.Nil(s.T(), err)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
