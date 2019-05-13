package testutils

import (
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	err := ImportCCLFPackage("dev", "test")
	assert.Nil(err)
	// MOAR TESTS HERE TO CHECK
}
