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

const ccflRefDateKey = "CCLF_REF_DATE"

var origDate string

func (s *CCLFUtilTestSuite) SetupSuite() {
	origDate = os.Getenv(ccflRefDateKey)
}

func (s *CCLFUtilTestSuite) SetupTest() {
	models.InitializeGormModels()
	os.Setenv(ccflRefDateKey, "D181201")
}

func (s *CCLFUtilTestSuite) TearDownSuite() {
	os.Setenv(ccflRefDateKey, origDate)
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFUtilTestSuite))
}

func (s *CCLFUtilTestSuite) TestImportInvalidSizeACO() {
	assert := assert.New(s.T())
	os.Setenv(ccflRefDateKey, "D190617")
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
	err := ImportCCLFPackage("large", "unit-test-new-beneficiaries")
	assert.Nil(err)
}

func (s *CCLFUtilTestSuite) TearDownTest() {
	err := os.RemoveAll(DestDir)
	if err != nil {
		fmt.Println("Failed to delete CCLF DestDir")
	}
}
