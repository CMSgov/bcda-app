package health

import (
	"database/sql"
	"os"

	"github.com/CMSgov/bcda-app/bcda/client"
)

func IsDatabaseOK() bool {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return false
	}
	defer db.Close()

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
