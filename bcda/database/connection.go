package database

import (
	"database/sql"
	"log"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
)

// Variable substitution to support testing.
var LogFatal = log.Fatal

func GetDbConnection() *sql.DB {
	databaseURL := conf.GetEnv("DATABASE_URL")
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

	db, err := sql.Open("pgx", dc.ConnectionString(databaseURL))
	if err != nil {
		LogFatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		LogFatal(pingErr)
	}

	return db
}
