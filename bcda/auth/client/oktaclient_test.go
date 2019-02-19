// +build okta

// To enable this test suite:
// 1. Put an appropriate token into env var OKTA_CLIENT_TOKEN
// 2. Put an existing Okta user email address into OKTA_EMAIL
// 3. Run "go test -tags=okta" from the bcda/auth/client directory

package client_test

import (
	"os"
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
	// The email in OKTA_EMAIL should represent a test user present in the Okta sandbox environment
	userEmail, success := os.LookupEnv("OKTA_EMAIL")
	assert.True(s.T(), success, "Please set OKTA_EMAIL to match a test user account")

	u, err := client.FindUser(userEmail)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), "", u)

	// Should return more than one user
	u, err = client.FindUser("user")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "", u)
}

func (s *OTestSuite) TestHealthCheck() {
	err := client.HealthCheck()
	assert.Nil(s.T(), err)
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
