package health

import (
	ssasClient "github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/log"
	_ "github.com/jackc/pgx"
)

func IsDatabaseOK() (result string, ok bool) {
	db := database.Connection
	if err := db.Ping(); err != nil {
		log.API.Error("Health check: database ping error: ", err.Error())
		return "database ping error", false
	}

	return "ok", true
}

func IsWorkerDatabaseOK() (result string, ok bool) {
	db := database.Connection
	if err := db.Ping(); err != nil {
		log.Worker.Error("Health check: database ping error: ", err.Error())
		return "database ping error", false
	}

	return "ok", true
}

func IsBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient(client.NewConfig("/v1/fhir"))
	if err != nil {
		log.Worker.Error("Health check: Blue Button client error: ", err.Error())
		return false
	}

	_, err = bbc.GetMetadata()
	if err != nil {
		log.Worker.Error("Health check: Blue Button connection error: ", err.Error())
		return false
	}

	return true
}

func IsSsasOK() (result string, ok bool) {
	c, err := ssasClient.NewSSASClient()
	if err != nil {
		log.Auth.Errorf("no client for SSAS. no provider set; %s", err.Error())
		return "No client for SSAS. no provider set", false
	}
	if err := c.GetHealth(); err != nil {
		log.API.Error("Health check: ssas health check error: ", err.Error())
		return "Cannot connect to SSAS", false
	}
	return "ok", true
}
