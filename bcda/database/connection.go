package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func GetGORMDbConnection() *gorm.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		LogFatal(err)
	}
	return db
}
