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

// func (s *DatabaseConfigSuite) TestLoadConfigFromParameterStoreSuccess() {
// 	assert := assert.New(s.T())

// 	env := uuid.NewUUID()
// 	cleanupEnv := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
// 		{Name: "ENV", Value: env.String()},
// 		{Name: "DATABASE_URL", Value: ""},
// 	})
// 	defer cleanupEnv()

// 	cleanupParams := testUtils.SetParameter(s.T(), fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env), "my-super-secure-database-url")
// 	// 	{Name: , Value: "my-super-secure-database-url", Type: "SecureString"},
// 	// })
// 	defer cleanupParams()

// 	cfg, err := LoadConfig()
// 	assert.Nil(err)
// 	assert.Equal("my-super-secure-database-url", cfg.DatabaseURL)
// }

// func (s *DatabaseConfigSuite) TestLoadConfigFromParameterStoreMissingDatabaseUrl() {
// 	assert := assert.New(s.T())

// 	env := uuid.NewUUID()
// 	cleanupEnv := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
// 		{Name: "ENV", Value: env.String()},
// 		{Name: "DATABASE_URL", Value: ""},
// 	})
// 	defer cleanupEnv()

// 	cleanupParams := testUtils.SetParameter(s.T(), "", "")
// 	defer cleanupParams()

// 	cfg, err := LoadConfig()
// 	assert.Nil(cfg)
// 	assert.Contains(err.Error(), fmt.Sprintf("invalid parameters error: /bcda/%s/api/DATABASE_URL", env))
// }
