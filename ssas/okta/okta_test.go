// +build okta

// To enable this test suite:
// Run "go test -tags=okta -v" from the ssas/okta directory
package okta

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"os"
	"regexp"
	"testing"
)

type OTestSuite struct {
	suite.Suite
}

func (s *OTestSuite) TestConfig() {
	originalOktaBaseUrl := os.Getenv("OKTA_CLIENT_ORGURL")
	originalOktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	os.Unsetenv("OKTA_CLIENT_ORGURL")
	os.Unsetenv("OKTA_CLIENT_TOKEN")

	err := config()
	require.NotNil(s.T(), err)
	assert.Regexp(s.T(), regexp.MustCompile("(OKTA_[A-Z_]*=, ){2}(OKTA_CLIENT_TOKEN=)"), err)

	os.Setenv("OKTA_CLIENT_TOKEN", originalOktaToken)

	err = config()
	assert.NotNil(s.T(), err)
	assert.Regexp(s.T(), regexp.MustCompile("(OKTA_[A-Z_]*=, ){2}(OKTA_CLIENT_TOKEN=\\[Redacted\\])"), err)

	os.Setenv("OKTA_CLIENT_ORGURL", originalOktaBaseUrl)
	os.Setenv("OKTA_CLIENT_TOKEN", originalOktaToken)

	err = config()
	assert.Nil(s.T(), err)
}

func (s *OTestSuite) TestParseOktaErrorSuccess() {
	oktaResponse := []byte(`{"errorCode":"E0000011","errorSummary":"Invalid token provided","errorLink":"E0000011","errorId":"oae3iIXhkQVQ2izGNwhnR47JQ","errorCauses":[]}`)
	oktaError, err := ParseOktaError(oktaResponse)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), oktaError)
	assert.Equal(s.T(), "Invalid token provided", oktaError.ErrorSummary)
}

func TestOTestSuite(t *testing.T) {
	suite.Run(t, new(OTestSuite))
}
