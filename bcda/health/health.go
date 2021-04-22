package health

import (
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/log"
	_ "github.com/jackc/pgx"
)

func IsDatabaseOK() bool {
	db := database.Connection
	if err := db.Ping(); err != nil {
		log.API.Error("Health check: database ping error: ", err.Error())
		return false
	}

	return true
}

func IsBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient(client.NewConfig("/v1/fhir"))
	if err != nil {
		log.API.Error("Health check: Blue Button client error: ", err.Error())
		return false
	}

	_, err = bbc.GetMetadata()
	if err != nil {
		log.API.Error("Health check: Blue Button connection error: ", err.Error())
		return false
	}

	return true
}
