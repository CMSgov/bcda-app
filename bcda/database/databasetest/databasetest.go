package databasetest

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/ccoveille/go-safecast"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var dsnPattern *regexp.Regexp = regexp.MustCompile(`(?P<conn>postgresql\:\/\/\S+\:\S+\@\S+\:\d+\/)(?P<dbname>.*)(?P<options>\?.*)`)

// CreateDatabase creates a clone of the database referenced by DATABASE_URL
// It returns the sql.DB connection, pgx pool connection, and the created database name
func CreateDatabase(t *testing.T, migrationPath string, cleanup bool) (*sql.DB, *pgxv5Pool.Pool, string) {
	cfg, err := database.LoadConfig()
	assert.NoError(t, err)
	dsn := cfg.DatabaseURL
	db := database.Connect()

	newDBName := strings.ReplaceAll(fmt.Sprintf("%s_%s", dbName(dsn), uuid.New()), "-", "_")
	newDSN := dsnPattern.ReplaceAllString(dsn, fmt.Sprintf("${conn}%s${options}", newDBName))

	// Use CREATE DATABASE + migrate to build tables instead of
	// CREATE DATABASE <NEW> WITH TEMPLATE <OLD>
	// the WITH TEMPLATE requires that there are no active connections to the old database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", newDBName))
	assert.NoError(t, err)
	setupBCDATables(t, migrationPath, newDSN)

	newDB, err := sql.Open("pgx", newDSN)
	assert.NoError(t, err)

	// Create pgx pool config with the test database URL
	pgxv5PoolConfig, err := pgxv5Pool.ParseConfig(newDSN)
	assert.NoError(t, err)

	maxConns, err := safecast.ToInt32(cfg.MaxOpenConns)
	assert.NoError(t, err)

	pgxv5PoolConfig.MaxConns = maxConns
	pgxv5PoolConfig.MaxConnIdleTime = time.Duration(cfg.ConnMaxIdleTime) * time.Second
	pgxv5PoolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute
	pgxv5PoolConfig.HealthCheckPeriod = time.Duration(cfg.HealthCheckSec) * time.Second

	newPool, err := pgxv5Pool.NewWithConfig(context.Background(), pgxv5PoolConfig)
	assert.NoError(t, err)

	if cleanup {
		t.Cleanup(func() {
			newPool.Close()
			assert.NoError(t, newDB.Close())
			_, err = db.Exec(fmt.Sprint("DROP DATABASE " + newDBName))
			assert.NoError(t, err)
		})
	}
	return newDB, newPool, newDBName
}

func dbName(dsn string) string {
	return dsnPattern.FindStringSubmatch(dsn)[2]
}

func setupBCDATables(t *testing.T, migrationPath, dsn string) {
	m, err := migrate.New("file://"+migrationPath, setMigrationsTable(dsn, "migrations_bcda"))
	assert.NoError(t, err)
	assert.NoError(t, m.Up())
	srcErr, dbErr := m.Close()
	assert.NoError(t, srcErr)
	assert.NoError(t, dbErr)
}

func setMigrationsTable(dsn, migrationsTable string) string {
	return dsnPattern.ReplaceAllString(dsn, fmt.Sprintf("${conn}${dbname}${options}&x-migrations-table=%s", migrationsTable))
}
