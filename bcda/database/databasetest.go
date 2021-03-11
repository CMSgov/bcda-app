package database

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var dsnPattern *regexp.Regexp = regexp.MustCompile(`(?P<conn>postgresql\:\/\/\S+\:\S+\@\S+\:\d+\/?)(?P<dbname>.*?)(?P<options>\?.*?)`)

// CreateDatabase creates a clone of the database referenced by DATABASE_URL
// It returns the connection to the database as well as the created name
func CreateDatabase(t *testing.T, cleanup bool) (*sql.DB, string) {
	dsn := conf.GetEnv("DATABASE_URL")
	db := getDbConnection(dsn)
	newDBName := strings.ReplaceAll(fmt.Sprintf("%s_%s", dbName(dsn), uuid.New()), "-", "_")

	_, err := db.Exec(fmt.Sprintf(`SELECT pg_terminate_backend(pg_stat_activity.pid) 
FROM pg_stat_activity 
WHERE pg_stat_activity.datname = '%s' 
AND pid <> pg_backend_pid();`, dbName(dsn)))
	assert.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s", newDBName, dbName(dsn)))
	assert.NoError(t, err)
	fmt.Printf("%s\n", newDBName)

	newDB := getDbConnection(dsnPattern.ReplaceAllString(dsn, fmt.Sprintf("${conn}%s${options}", newDBName)))
	if cleanup {
		t.Cleanup(func() {
			assert.NoError(t, newDB.Close())
			_, err = db.Exec(fmt.Sprintf("DROP DATABASE " + newDBName))
			assert.NoError(t, err)
			db.Close()
		})
	}
	return newDB, newDBName
}

func dbName(dsn string) string {
	return dsnPattern.FindStringSubmatch(dsn)[2]
}
