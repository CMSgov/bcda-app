package testutils

import (
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CCLFUtilTestSuite struct {
	suite.Suite
}

var origDate string

func (s *CCLFUtilTestSuite) SetupSuite() {
	origDate = conf.GetEnv("CCLF_REF_DATE")
}

func (s *CCLFUtilTestSuite) SetupTest() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "D181201")
}

func (s *CCLFUtilTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", origDate)
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFUtilTestSuite))
}

func (s *CCLFUtilTestSuite) TestImportInvalidSizeACO() {
	assert := assert.New(s.T())
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "D190617")
	err := ImportCCLFPackage("NOTREAL", "test", models.FileTypeDefault)
	assert.EqualError(err, "invalid argument for ACO size")
}

func (s *CCLFUtilTestSuite) TestImportInvalidEnvironment() {
	assert := assert.New(s.T())
	err := ImportCCLFPackage("dev", "environment", models.FileTypeDefault)
	assert.EqualError(err, "invalid argument for environment")
}

func (s *CCLFUtilTestSuite) TestImport() {
	tests := []struct {
		env     string
		acoSize string
	}{
		// We should only have one entry for a given acoSize.
		// If there are duplicates, we may collide on the resulting CCLF file name.
		{"test", "dev"},
		{"test-new-beneficiaries", "dev-cec"},
		{"test", "dev-ng"},
		{"test", "dev-ckcc"},
		{"test-new-beneficiaries", "dev-kcf"},
		{"test", "dev-dc"},
	}
	for _, tt := range tests {
		for _, fileType := range []models.CCLFFileType{models.FileTypeDefault, models.FileTypeRunout} {
			s.T().Run(fmt.Sprintf("ACO Size %s - Env %s - File Type %s", tt.acoSize, tt.env, fileType),
				func(t *testing.T) {
					err := ImportCCLFPackage(tt.acoSize, tt.env, fileType)
					assert.NoError(t, err)
				})
		}
	}
}

func (s *CCLFUtilTestSuite) TestHasAnyPrefix() {
	tests := []struct {
		s        string
		prefixes []string
		found    bool
	}{
		{"A0001", []string{"A"}, true},
		{"A0001", []string{"B"}, false},
		{"Z0001", []string{"X", "Y", "Z"}, true},
		{"Z0001", []string{"A", "B", "C"}, false},
	}
	for _, tt := range tests {
		s.T().Run(fmt.Sprintf("Test String %s - Prefix(es) %s - Expect to be found %t", tt.s, tt.prefixes, tt.found),
			func(t *testing.T) {
				assert.Equal(t, tt.found, hasAnyPrefix(tt.s, tt.prefixes...))
			})
	}
}
