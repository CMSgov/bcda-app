package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

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

	err = s.server.SetSigningKeys(os.Getenv("SSAS_SERVER_TEST_PRIVATE_KEY"))
	assert.Nil(s.T(), err)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
