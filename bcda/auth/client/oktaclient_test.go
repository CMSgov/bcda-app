// +build okta

// To enable this test suite:
// 1. Put an appropriate token into env var OKTA_CLIENT_TOKEN
// 2. Put an existing Okta user email address into OKTA_EMAIL
// 3. Run "go test -tags=okta -v" from the bcda/auth/client directory

package client_test

import (
	"testing"

	"github.com/okta/okta-sdk-golang/okta"
	"github.com/stretchr/testify/suite"
)

type OTestSuite struct {
	suite.Suite
	oClient *okta.Client
}

func (s *OTestSuite) TearDownTest() {
}

func TestOTestSuite(t *testing.T) {
	suite.Run(t, new(OTestSuite))
}
