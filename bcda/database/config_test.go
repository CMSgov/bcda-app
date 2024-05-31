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
		{Name: "DATABASE_URL", Value: "my-super-secure-database-url"},
		{Name: "QUEUE_DATABASE_URL", Value: "my-super-secure-queue-database-url"},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	assert.Nil(err)
	assert.Equal("my-super-secure-database-url", cfg.DatabaseURL)
	assert.Equal("my-super-secure-queue-database-url", cfg.QueueDatabaseURL)
}

func (s *DatabaseConfigSuite) TestLoadConfigMissingDatabaseUrl() {
	assert := assert.New(s.T())

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "DATABASE_URL", Value: ""},
		{Name: "QUEUE_DATABASE_URL", Value: "my-super-secure-queue-database-url"},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	assert.Nil(cfg)
	assert.Contains(err.Error(), "invalid config, DatabaseURL must be set")
}

func (s *DatabaseConfigSuite) TestLoadConfigMissingQueueDatabaseUrl() {
	assert := assert.New(s.T())

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "DATABASE_URL", Value: "my-super-secure-database-url"},
		{Name: "QUEUE_DATABASE_URL", Value: ""},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	assert.Nil(cfg)
	assert.Contains(err.Error(), "invalid config, QueueDatabaseURL must be set")
}

func (s *DatabaseConfigSuite) TestLoadConfigFromParameterStoreSuccess() {
	assert := assert.New(s.T())

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/bcda/local/api/DATABASE_URL", Value: "my-super-secure-database-url", Type: "SecureString"},
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "my-super-secure-queue-database-url", Type: "SecureString"},
	})
	defer cleanupParams()

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "DATABASE_URL", Value: ""},
		{Name: "QUEUE_DATABASE_URL", Value: ""},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfigFromParameterStore("/bcda/local/api/DATABASE_URL", "/bcda/local/api/QUEUE_DATABASE_URL")
	assert.Nil(err)
	assert.Equal("my-super-secure-database-url", cfg.DatabaseURL)
	assert.Equal("my-super-secure-queue-database-url", cfg.QueueDatabaseURL)
}

func (s *DatabaseConfigSuite) TestLoadConfigFromParameterStoreMissingDatabaseUrl() {
	assert := assert.New(s.T())

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "my-super-secure-queue-database-url", Type: "SecureString"},
	})
	defer cleanupParams()

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "DATABASE_URL", Value: ""},
		{Name: "QUEUE_DATABASE_URL", Value: ""},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfigFromParameterStore("/bcda/local/api/DATABASE_URL", "/bcda/local/api/QUEUE_DATABASE_URL")

	assert.Nil(cfg)
	assert.Contains(err.Error(), "invalid parameters error: /bcda/local/api/DATABASE_URL")
}

func (s *DatabaseConfigSuite) TestLoadConfigFromParameterStoreMissingQueueDatabaseUrl() {
	assert := assert.New(s.T())

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/bcda/local/api/DATABASE_URL", Value: "my-super-secure-database-url", Type: "SecureString"},
	})
	defer cleanupParams()

	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "DATABASE_URL", Value: ""},
		{Name: "QUEUE_DATABASE_URL", Value: ""},
	})
	defer cleanupEnvVars()

	cfg, err := LoadConfigFromParameterStore("/bcda/local/api/DATABASE_URL", "/bcda/local/api/QUEUE_DATABASE_URL")

	assert.Nil(cfg)
	assert.Contains(err.Error(), "invalid parameters error: /bcda/local/api/QUEUE_DATABASE_URL")
}
