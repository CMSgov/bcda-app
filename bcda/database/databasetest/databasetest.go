package databasetest

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var dsnPattern *regexp.Regexp = regexp.MustCompile(`(?P<conn>postgresql\:\/\/\S+\:\S+\@\S+\:\d+\/)(?P<dbname>.*)(?P<options>\?.*)`)

// CreateDatabase creates a clone of the database referenced by DATABASE_URL
// It returns the connection to the database as well as the created name
func CreateDatabase(t *testing.T, migrationPath string, cleanup bool) (*sql.DB, string) {
	dsn := conf.GetEnv("DATABASE_URL")
	db := database.GetDbConnectionDSN(dsn)

	newDBName := strings.ReplaceAll(fmt.Sprintf("%s_%s", dbName(dsn), uuid.New()), "-", "_")
	newDSN := dsnPattern.ReplaceAllString(dsn, fmt.Sprintf("${conn}%s${options}", newDBName))

	// Use CREATE DATABASE + migrate to build tables instead of
	// CREATE DATABASE <NEW> WITH TEMPLATE <OLD>
	// the WITH TEMPLATE requires that there are no active connections to the old database
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", newDBName))
	assert.NoError(t, err)
	setupBCDATables(t, migrationPath, newDSN)

	newDB := database.GetDbConnectionDSN(newDSN)
	if cleanup {
		t.Cleanup(func() {
			assert.NoError(t, newDB.Close())
			_, err = db.Exec(fmt.Sprintf("DROP DATABASE " + newDBName))
			assert.NoError(t, err)
			assert.NoError(t, db.Close())
		})
	}
	return newDB, newDBName
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
