package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func GetDbConnection() *sql.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	return db
}
