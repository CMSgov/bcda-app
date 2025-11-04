package database

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DatabaseConfigSuite struct {
	suite.Suite
}

func TestDatabaseConfigSuite(t *testing.T) {
	suite.Run(t, new(DatabaseConfigSuite))
}

func (s *DatabaseConfigSuite) TestLoadConfigSuccess() {
	assert := assert.New(s.T())

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "ENV", Value: ""},
		{Name: "DATABASE_URL", Value: "my-super-secure-database-url"},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	assert.Nil(err)
	assert.Equal("my-super-secure-database-url", cfg.DatabaseURL)
}

func (s *DatabaseConfigSuite) TestLoadConfigMissingDatabaseUrl() {
	assert := assert.New(s.T())

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "ENV", Value: ""},
		{Name: "DATABASE_URL", Value: ""},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	assert.Nil(cfg)
	assert.Contains(err.Error(), "invalid config, DatabaseURL must be set")
}
