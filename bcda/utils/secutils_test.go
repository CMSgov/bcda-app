package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"

    configuration "github.com/CMSgov/bcda-app/config"
)

type SecutilsTestSuite struct {
	suite.Suite
}

//
//func (s *SecutilsTestSuite) SetupTest() {
//	publicKeyFile := configuration.GetEnv("ATO_PUBLIC_KEY_FILE")
//	privateKeyFile := configuration.GetEnv("ATO_PRIVATE_KEY_FILE")
//}

func (s *SecutilsTestSuite) TestOpenPrivateKeyFile() {
	atoPrivateKeyFile, err := os.Open(configuration.GetEnv("ATO_PRIVATE_KEY_FILE"))
	if err != nil {
		panic(err)
	}

	assert.NotNil(s.T(), OpenPrivateKeyFile(atoPrivateKeyFile))
}

func (s *SecutilsTestSuite) TestOpenPublicKeyFile() {
	atoPublicKeyFile, err := os.Open(configuration.GetEnv("ATO_PUBLIC_KEY_FILE"))
	if err != nil {
		fmt.Println("failed to open file")
		panic(err)
	}

	assert.NotNil(s.T(), OpenPublicKeyFile(atoPublicKeyFile))
}

func TestSecutilsTestSuite(t *testing.T) {
	suite.Run(t, new(SecutilsTestSuite))
}
