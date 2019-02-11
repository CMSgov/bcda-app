package client_test

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/okta/okta-sdk-golang/okta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OTestSuite struct {
	suite.Suite
	oClient *okta.Client
}

func (s *OTestSuite) SetupTest() {
	s.oClient = client.NewOktaClient()
}

func (s *OTestSuite) TestFindUser() {
	// This test user should be present in the Okta sandbox environment
	u, err := client.FindUser("shawn@bcda.aco-group.us")
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), "", u)

	// Should return more than one user
	u, err = client.FindUser("user")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "", u)
}

func (s *OTestSuite) TestHealthCheck() {
	b, err := client.HealthCheck()
	assert.Nil(s.T(), err)
	assert.True(s.T(), b)
}

func (s *OTestSuite) TestDeleteUser() {
	// Real user ID's are random lowercase alpha/numeric
	b, err := client.DeleteUser("INVALID_USER_ID")
	assert.NotNil(s.T(), err)
	assert.False(s.T(), b)
}

func (s *OTestSuite) TearDownTest() {
}

func TestOTestSuite(t *testing.T) {
	suite.Run(t, new(OTestSuite))
}
