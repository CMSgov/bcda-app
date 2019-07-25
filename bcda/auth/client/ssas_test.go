package client_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SSASClientTestSuite struct {
	suite.Suite
}

func (s *SSASClientTestSuite) TestNewSSASClient() {}

func (s *SSASClientTestSuite) TestCreateSystem() {}

func (s *SSASClientTestSuite) TestGetPublicKey() {}

func (s *SSASClientTestSuite) TestResetCredentials() {}

func (s *SSASClientTestSuite) TestDeleteCredentials() {}

func (s *SSASClientTestSuite) TestSSASClientTestSuite(t *testing.T) {
	suite.Run(t, new(SSASClientTestSuite))
}
