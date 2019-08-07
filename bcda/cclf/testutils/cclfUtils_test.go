package testutils

import (
	"fmt"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	assert.EqualError(err, "invalid argument for ACO size")
}

func (s *CCLFUtilTestSuite) TestImportInvalidEnvironment() {
	assert := assert.New(s.T())
	err := ImportCCLFPackage("dev", "environment")
	assert.EqualError(err, "invalid argument for environment")
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
