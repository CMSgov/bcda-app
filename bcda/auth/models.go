package auth

import (
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

func GetACO(col, val string) (models.ACO, error) {
	var (
		db  = database.GetGORMDbConnection()
		aco models.ACO
		err error
	)
	defer database.Close(db)

	if err = db.First(&aco, col+" = ?", val).Error; err != nil {
		err = fmt.Errorf("no ACO record found for %s", val)
	}
	return aco, err
}

func GetACOByUUID(id string) (models.ACO, error) {
	return GetACO("uuid", id)
}

func GetACOByClientID(clientID string) (models.ACO, error) {
	return GetACO("client_id", clientID)
}

func GetACOByCMSID(cmsID string) (models.ACO, error) {
	return GetACO("cms_id", cmsID)
}
