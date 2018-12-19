package secutils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type SecutilsTestSuite struct {
	suite.Suite
}
//
//func (s *SecutilsTestSuite) SetupTest() {
//	publicKeyFile := os.Getenv("ATO_PUBLIC_KEY_FILE")
//	privateKeyFile := os.Getenv("ATO_PRIVATE_KEY_FILE")
//}

func (s *SecutilsTestSuite) TestOpenPrivateKeyFile() {
	atoPrivateKeyFile, err := os.Open(os.Getenv("ATO_PRIVATE_KEY_FILE"))
	if err != nil {
		panic(err)
	}

	assert.NotNil(s.T(), OpenPrivateKeyFile(atoPrivateKeyFile))
}

func (s *SecutilsTestSuite) TestOpenPublicKeyFile() {
	atoPublicKeyFile, err := os.Open(os.Getenv("ATO_PUBLIC_KEY_FILE"))
	if err != nil {
		fmt.Println("failed to open file")
		panic(err)
	}

	assert.NotNil(s.T(), OpenPublicKeyFile(atoPublicKeyFile))
}

func TestSecutilsTestSuite(t *testing.T) {
	suite.Run(t, new(SecutilsTestSuite))
}


