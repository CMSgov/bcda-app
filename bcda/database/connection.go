package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/bgentry/que-go"
	"github.com/ccoveille/go-safecast"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/jackc/pgx/stdlib"

	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/CMSgov/bcda-app/bcda/nrpgx"
	"github.com/sirupsen/logrus"
)

var (
	Connection      *sql.DB
	QueueConnection *pgx.ConnPool
	Pgxv5Connection *pgxv5Pool.Pool
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

	Pgxv5Connection, err = createPgxv5DB(cfg)
	if err != nil {
		logrus.Fatalf("Failed to create pgxv5 DB connection %s", err.Error())
	}

	startHealthCheck(
		context.Background(),
		Connection,
		QueueConnection,
		Pgxv5Connection,
		time.Duration(cfg.HealthCheckSec)*time.Second,
	)
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

	db, err := sql.Open("nrpgx", dc.ConnectionString(cfg.DatabaseURL))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

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

func createPgxv5DB(cfg *Config) (*pgxv5Pool.Pool, error) {
	ctx := context.Background()

	pgxv5PoolConfig, err := pgxv5Pool.ParseConfig((cfg.DatabaseURL))
	if err != nil {
		return nil, err
	}

	maxConns, err := safecast.ToInt32(cfg.MaxOpenConns)
	if err != nil {
		return nil, err
	}

	pgxv5PoolConfig.MaxConns = maxConns
	pgxv5PoolConfig.MaxConnIdleTime = time.Duration(cfg.ConnMaxIdleTime)
	pgxv5PoolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetimeMin)
	pgxv5PoolConfig.HealthCheckPeriod = time.Duration(cfg.HealthCheckSec)

	dbPool, err := pgxv5Pool.NewWithConfig(ctx, pgxv5PoolConfig)
	if err != nil {
		panic(err)
	}

	return dbPool, err
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
func startHealthCheck(ctx context.Context, db *sql.DB, pool *pgx.ConnPool, pgxv5Pool *pgxv5Pool.Pool, interval time.Duration) {
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

				// Handle acquiring connection, pinging, and releasing App DB connection
				if err := db.Ping(); err != nil {
					logrus.Warnf("Failed to ping %s", err.Error())
				}

				// Aquire and ping Queue DB
				c, err := pool.Acquire()
				if err != nil {
					logrus.Warnf("Failed to acquire Queue DB connection %s", err.Error())
					continue
				}
				if err := c.Ping(context.Background()); err != nil {
					logrus.Warnf("Failed to ping Queue DB %s", err.Error())
				}
				pool.Release(c)

				// Aquire and ping pgxv5 connection to App DB
				pgxv5Conn, err := pgxv5Pool.Acquire(ctx)
				if err != nil {
					logrus.Warnf("Failed to acquire pgxv5 App DB connection: %s", err.Error())
					continue
				}
				if err := pgxv5Conn.Ping(ctx); err != nil {
					logrus.Warnf("Failed to ping pgxv5 App DB: %s", err.Error())
				}
				pgxv5Conn.Release()
			}
		}
	}()
}
