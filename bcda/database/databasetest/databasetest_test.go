package databasetest

import (
	"database/sql"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	_ "github.com/CMSgov/bcda-app/bcda/nrpgx"
	"github.com/stretchr/testify/assert"
)

func TestCreateDatabase(t *testing.T) {
	var (
		dropped    string
		notDropped string
	)
	// Run in sub test to verify database is dropped
	t.Run("CreateAndDrop", func(sub *testing.T) {
		var db *sql.DB
		db, dropped = CreateDatabase(sub, "../../../db/migrations/bcda/", true)
		assert.NotNil(t, db)
		assert.NoError(t, db.Close())
	})
	t.Run("CreateAndNoDrop", func(sub *testing.T) {
		var db *sql.DB
		db, notDropped = CreateDatabase(sub, "../../../db/migrations/bcda/", false)
		assert.NotNil(t, db)
		assert.NoError(t, db.Close())
	})

	db := database.Connect()

	var count int
	assert.NoError(t,
		db.QueryRow("SELECT COUNT(1) FROM pg_database WHERE datname = $1", dropped).
			Scan(&count))
	assert.Equal(t, 0, count, "Database should've been dropped")
	assert.NoError(t,
		db.QueryRow("SELECT COUNT(1) FROM pg_database WHERE datname = $1", notDropped).
			Scan(&count))
	assert.Equal(t, 1, count, "Database should not have been dropped")
}
