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
			log.Info("failed to close db connection at bcda/health/health.go#IsDatabaseOK() because %s", err)
		}
	}()
	if err != nil {
		return false
	}

	return db.Ping() == nil
}

func IsBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient()
	if err != nil {
		return false
	}

	_, err = bbc.GetMetadata()
	return err == nil
}
