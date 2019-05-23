package testutils

import (
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type CCLFUtilTestSuite struct {
	suite.Suite
}

func (s *CCLFUtilTestSuite) SetupTest() {
	models.InitializeGormModels()
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFUtilTestSuite))
}

func (s *CCLFUtilTestSuite) TestImportInvalidSizeACO() {
	assert := assert.New(s.T())
	err := ImportCCLFPackage("NOTREAL", "test")
	assert.NotNil(err)
}

func (s *CCLFUtilTestSuite) TestImportInvalidEnvironment() {
	assert := assert.New(s.T())
	err := ImportCCLFPackage("dev", "environment")
	assert.NotNil(err)
}

func (s *CCLFUtilTestSuite) TestImport() {
	assert := assert.New(s.T())
	err := ImportCCLFPackage("dev", "unit-test")
	assert.Nil(err)
}

func (s *CCLFUtilTestSuite) TearDownTest() {
	err := os.RemoveAll(DestDir)
	if err != nil {
		fmt.Println("Failed to delete CCLF DestDir")
	}
}
