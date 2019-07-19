package auth_test

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/stretchr/testify/suite"
)

type SSASPluginTestSuite struct {
	suite.Suite
	p auth.SSASPlugin
}

func (s *SSASPluginTestSuite) SetupSuite() {
	s.p = auth.SSASPlugin{}
}

func (s *SSASPluginTestSuite) TestRegisterSystem() {

}

func (s *SSASPluginTestSuite) TestUpdateSystem() {

}

func (s *SSASPluginTestSuite) TestDeleteSystem() {

}

func (s *SSASPluginTestSuite) TestResetSecret() {

}

func (s *SSASPluginTestSuite) TestRevokeSystemCredentials() {

}

func (s *SSASPluginTestSuite) TestMakeAccessToken() {

}

func (s *SSASPluginTestSuite) TestRevokeAccessToken() {

}

func (s *SSASPluginTestSuite) TestAuthorizeAccess() {

}

func (s *SSASPluginTestSuite) TestVerifyToken() {

}

func TestSSASPluginSuite(t *testing.T) {
	suite.Run(t, new(SSASPluginTestSuite))
}
