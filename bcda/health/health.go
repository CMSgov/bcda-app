package health

import (
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	_ "github.com/jackc/pgx"

	log "github.com/sirupsen/logrus"
)

func IsDatabaseOK() bool {
	db := database.Connection
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
