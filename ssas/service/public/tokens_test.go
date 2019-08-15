package public

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/ssas/service"
)

type PublicTokenTestSuite struct {
	suite.Suite
	server *service.Server
}

func (s *PublicTokenTestSuite) SetupSuite() {
	info := make(map[string][]string)
	info["public"] = []string{"token", "register"}
	s.server = Server()
	err := os.Setenv("DEBUG", "true")
	assert.Nil(s.T(), err)
}

func (s *PublicTokenTestSuite) TestMintMFAToken() {
	token, ts, err := MintMFAToken("my_okta_id")

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)
}

func (s *PublicTokenTestSuite) TestMintMFATokenMissingID() {
	token, ts, err := MintMFAToken("")

	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), token)
	assert.Equal(s.T(),"", ts)
}

func (s *PublicTokenTestSuite) TestMintRegistrationToken() {
	groupIDs := []string{"A0000", "A0001"}
	token, ts, err := MintRegistrationToken("my_okta_id", groupIDs)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)
}

func (s *PublicTokenTestSuite) TestMintRegistrationTokenMissingID() {
	groupIDs := []string{"", ""}
	token, ts, err := MintRegistrationToken("my_okta_id", groupIDs)

	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), token)
	assert.Equal(s.T(), "", ts)
}

func (s *PublicTokenTestSuite) TestEmpty() {
	groupIDs := []string{"", ""}
	assert.True(s.T(), empty(groupIDs))

	groupIDs = []string{"", "asdf"}
	assert.False(s.T(), empty(groupIDs))
}

func TestPublicTokenTestSuite(t *testing.T) {
	suite.Run(t, new(PublicTokenTestSuite))
}