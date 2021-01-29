package health

import (
	"database/sql"
	"os"

	"github.com/CMSgov/bcda-app/bcda/client"
	_ "github.com/jackc/pgx/stdlib"

	log "github.com/sirupsen/logrus"
)

func IsDatabaseOK() bool {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error("Health check: database connection error: ", err.Error())
		return false
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Infof("failed to close db connection at bcda/health/health.go#IsDatabaseOK() because %s", err)
		}
	}()

	if err := db.Ping(); err != nil {
		log.Error("Health check: database ping error: ", err.Error())
		return false
	}

	return true
}

func IsBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient(client.NewConfig("/v1/fhir"))
	if err != nil {
		log.Error("Health check: Blue Button client error: ", err.Error())
		return false
	}

	_, err = bbc.GetMetadata()
	if err != nil {
		log.Error("Health check: Blue Button connection error: ", err.Error())
		return false
	}

	return true
}
