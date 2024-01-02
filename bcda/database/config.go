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

	DatabaseURL      string `conf:"DATABASE_URL"`
	QueueDatabaseURL string `conf:"QUEUE_DATABASE_URL"`

	HealthCheckSec int `conf:"DB_HEALTH_CHECK_INTERVAL" conf_default:"5"`
}

func LoadConfig() (cfg *Config, err error) {
	cfg = &Config{}
	if err := conf.Checkout(cfg); err != nil {
		return nil, err
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("invalid config, DatabaseURL must be set")
	}
	if cfg.QueueDatabaseURL == "" {
		return nil, errors.New("invalid config, QueueDatabaseURL must be set")
	}

	log.API.Info("Successfully loaded configuration for Database.")

	return cfg, nil
}
