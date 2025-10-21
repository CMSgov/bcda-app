package database

import (
	"errors"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

type Config struct {
	MaxOpenConns       int `conf:"BCDA_DB_MAX_OPEN_CONNS" conf_default:"60"`
	MaxIdleConns       int `conf:"BCDA_DB_MAX_IDLE_CONNS" conf_default:"40"`
	ConnMaxLifetimeMin int `conf:"BCDA_DB_CONN_MAX_LIFETIME_MIN" conf_default:"5"`
	ConnMaxIdleTime    int `conf:"BCDA_DB_CONN_MAX_IDLE_TIME" conf_default:"30"`

	DatabaseURL string `conf:"DATABASE_URL"`

	HealthCheckSec int `conf:"DB_HEALTH_CHECK_INTERVAL" conf_default:"5"`
}

// Loads database URLs from environment variables.
func LoadConfig() (cfg *Config, err error) {
	cfg = &Config{}
	if err := conf.Checkout(cfg); err != nil {
		return nil, err
	}

	// if cfg.DatabaseURL == "" {
	// 	// Attempt to load database config from parameter store if ENV var is set.
	// 	// This generally indicates that we are running within our lambda environment.
	// 	env := os.Getenv("ENV")

	// 	if env != "" {
	// 		cfg, err = LoadConfigFromParameterStore(
	// 			fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env))

	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }

	if cfg.DatabaseURL == "" {
		return nil, errors.New("invalid config, DatabaseURL must be set")
	}

	log.API.Info("Successfully loaded configuration for Database.")
	return cfg, nil
}

// Loads database URL from parameter store instead of from environment variables.
// func LoadConfigFromParameterStore(dbUrlKey string) (cfg *Config, err error) {
// 	cfg = &Config{}
// 	if err := conf.Checkout(cfg); err != nil {
// 		return nil, err
// 	}

// 	// bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	cfg, err := config.LoadDefaultConfig(ctx)
// 	if err != nil {
// 		return awsParams{}, err
// 	}
// 	ssmClient := ssm.NewFromConfig(cfg)

// 	params, err := bcdaaws.GetParameters(ctx, ssmClient, paramNames)
// 	if err != nil {
// 		return awsParams{}, err
// 	}

// 	params, err := bcdaaws.GetParameters(bcdaSession, []*string{&dbUrlKey})
// 	if err != nil {
// 		return nil, err
// 	}

// 	cfg.DatabaseURL = params[dbUrlKey]

// 	return cfg, nil
// }
