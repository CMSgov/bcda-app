package health

import (
	"database/sql"
	"os"

	"github.com/CMSgov/bcda-app/bcda/client"

	log "github.com/sirupsen/logrus"
)

func IsDatabaseOK() bool {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	defer func() {
		if err := db.Close(); err != nil {
			log.Infof("failed to close db connection at bcda/health/health.go#IsDatabaseOK() because %s", err)
		}
	}()
	if err != nil {
		log.Error("Database connection check encountered an error:", err.Error())
		return false
	}

	if db.Ping() != nil {
		log.Error("Database connection check could not reach database")
		return false
	}

	return true
}

func IsBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient()
	if err != nil {
		log.Error("Blue Button connection check could not create client due to error:", err.Error())
		return false
	}

	_, err = bbc.GetMetadata()
	if err != nil {
		log.Error("Blue Button connection check encountered an error:", err.Error())
		return false
	}

	return true
}
