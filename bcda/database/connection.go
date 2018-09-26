package database

import (
	"database/sql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/lib/pq"
	"log"
	"os"
)

// Variable substitution to support testing.
var logFatal = log.Fatal

func GetDbConnection() *sql.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	db, _ := sql.Open("postgres", databaseURL)
	pingErr := db.Ping()
	if pingErr != nil {
		logFatal(pingErr)
	}
	return db
}

func GetGORMDbConnection() *gorm.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	db, _ := gorm.Open("postgres", databaseURL)
	pingErr := db.DB().Ping()
	if pingErr != nil {
		logFatal(pingErr)
	}
	return db
}
