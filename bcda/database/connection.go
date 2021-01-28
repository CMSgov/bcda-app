package database

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Variable substitution to support testing.
var LogFatal = log.Fatal

func GetDbConnection() *sql.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		LogFatal(err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		LogFatal(pingErr)
	}
	return db
}

// StartHealthCheck verifies the liveliness of the connections found in the supplied pool
//
// With que-go locked to pgx v3, we need a mechanism that will allow us to
// discard bad connections in the pgxpool (see: https://github.com/jackc/pgx/issues/494)
// This implementation is based off of the "fix" that is present in v4
// (see: https://github.com/jackc/pgx/blob/v4.10.0/pgxpool/pool.go#L333)
//
// StartHealthCheck returns immediately with the health check running in a goroutine that
// can be stopped via the supplied context
func StartHealthCheck(ctx context.Context, pool *pgx.ConnPool, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logrus.StandardLogger().Debug("Stopping pgxpool checker")
				return
			case <-ticker.C:
				logrus.StandardLogger().Debug("Sending ping")
				c, err := pool.Acquire()
				if err != nil {
					logrus.StandardLogger().Warnf("Failed to acquire connection %s", err.Error())
				}
				if err := c.Ping(ctx); err != nil {
					logrus.StandardLogger().Warnf("Failed to ping %s", err.Error())
				}
				pool.Release(c)
			}
		}
	}()
}
