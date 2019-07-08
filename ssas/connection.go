package ssas

import (
	"database/sql"
	"log"
	"os"
	"runtime"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/lib/pq"
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
	db, err := gorm.Open("postgres", databaseURL)
	if err != nil {
		LogFatal(err)
	}
	pingErr := db.DB().Ping()
	if pingErr != nil {
		LogFatal(pingErr)
	}
	return db
}

func Close(db *gorm.DB) {
	if err := db.Close(); err != nil {
		_, file, line, _ := runtime.Caller(1)
		Logger.Infof("failed to close db connection at %s#%d because %s", file, line, err)
	}
}
