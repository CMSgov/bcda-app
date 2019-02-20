// +build okta

// To enable this test suite:
// 1. Put an appropriate token into env var OKTA_CLIENT_TOKEN
// 2. Put an existing Okta user email address into OKTA_EMAIL
// 3. Run "go test -tags=okta -v" from the bcda/auth/client directory

package client

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/okta/okta-sdk-golang/okta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OTestSuite struct {
	suite.Suite
	oClient *okta.Client
}

func (s *OTestSuite) TestConfig() {
	originalOktaBaseUrl := os.Getenv("OKTA_CLIENT_ORGURL")
	originalOktaServerID := os.Getenv("OKTA_OAUTH_SERVER_ID")
	originalOktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	os.Unsetenv("OKTA_CLIENT_ORGURL")
	os.Unsetenv("OKTA_OAUTH_SERVER_ID")
	os.Unsetenv("OKTA_CLIENT_TOKEN")

	err := config()
	require.NotNil(s.T(), err)
	assert.Regexp(s.T(), regexp.MustCompile("(OKTA_[A-Z_]*=, ){2}(OKTA_CLIENT_TOKEN=)"), err)

	os.Setenv("OKTA_CLIENT_TOKEN", originalOktaToken)

	err = config()
	assert.NotNil(s.T(), err)
	fmt.Println(err)
	assert.Regexp(s.T(), regexp.MustCompile("(OKTA_[A-Z_]*=, ){2}(OKTA_CLIENT_TOKEN=\\[Redacted\\])"), err)

	os.Setenv("OKTA_CLIENT_ORGURL", originalOktaBaseUrl)
	os.Setenv("OKTA_OAUTH_SERVER_ID", originalOktaServerID)
	os.Setenv("OKTA_CLIENT_TOKEN", originalOktaToken)

	err = config()
	assert.Nil(s.T(), err)
}

func (s *OTestSuite) TearDownTest() {
}

func TestOTestSuite(t *testing.T) {
	suite.Run(t, new(OTestSuite))
}
