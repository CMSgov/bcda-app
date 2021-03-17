package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
)

var (
	Connection      *sql.DB
	QueueConnection *pgx.ConnPool
)

func init() {
	cfg, err := LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load database config %s", err.Error())
	}

	Connection, err = createDB(cfg)
	if err != nil {
		logrus.Fatalf("Failed to create db %s", err.Error())
	}

	QueueConnection, err = createQueue(cfg)
	if err != nil {
		logrus.Fatalf("Failed to create queue %s", err.Error())
	}

	startHealthCheck(context.Background(), Connection, QueueConnection,
		time.Duration(cfg.HealthCheckSec)*time.Second)
}

func createDB(cfg *Config) (*sql.DB, error) {
	dc := stdlib.DriverConfig{
		ConnConfig: pgx.ConnConfig{
			Logger:   logrusadapter.NewLogger(logrus.StandardLogger()),
			LogLevel: pgx.LogLevelError,
		},
		AfterConnect: func(c *pgx.Conn) error {
			// Can be used to ensure temp tables, indexes, etc. exist
			return nil
		},
	}

	stdlib.RegisterDriverConfig(&dc)

	db, err := sql.Open("pgx", dc.ConnectionString(cfg.DatabaseURL))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func createQueue(cfg *Config) (*pgx.ConnPool, error) {
	pgxCfg, err := pgx.ParseURI(cfg.QueueDatabaseURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     pgxCfg,
		MaxConnections: cfg.MaxOpenConns,
		// Needed to ensure the prepared statements are available for each connection.
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		return nil, err
	}

	return pool, err
}

// startHealthCheck verifies the liveliness of the connections found in the supplied pool
//
// With que-go locked to pgx v3, we need a mechanism that will allow us to
// discard bad connections in the pgxpool (see: https://github.com/jackc/pgx/issues/494)
// This implementation is based off of the "fix" that is present in v4
// (see: https://github.com/jackc/pgx/blob/v4.10.0/pgxpool/pool.go#L333)
//
// startHealthCheck returns immediately with the health check running in a goroutine that
// can be stopped via the supplied context
func startHealthCheck(ctx context.Context, db *sql.DB, pool *pgx.ConnPool, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logrus.Debug("Stopping health checker")
				return
			case <-ticker.C:
				logrus.StandardLogger().Debug("Sending ping")
				c, err := pool.Acquire()
				if err != nil {
					logrus.Warnf("Failed to acquire connection %s", err.Error())
					continue
				}
				if err := c.Ping(context.Background()); err != nil {
					logrus.Warnf("Failed to ping %s", err.Error())
				}
				pool.Release(c)

				// Handle acquiring connection, pinging, and releasing connection
				if err := db.Ping(); err != nil {
					logrus.Warnf("Failed to ping %s", err.Error())
				}
			}
		}
	}()
}
