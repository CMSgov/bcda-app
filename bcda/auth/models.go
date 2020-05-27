package auth

import (
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"
)

func InitializeGormModels() *gorm.DB {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// Migrate the schema
	// Add your new models here
	db.AutoMigrate()

	return db
}

type Token struct {
	UUID        uuid.UUID `json:"uuid"`   // uuid (primary key)
	Active      bool      `json:"active"` // active
	ACOID       uuid.UUID `json:"aco_id"`
	IssuedAt    int64     `json:"issued_at"`  // standard token claim; unix date
	ExpiresOn   int64     `json:"expires_on"` // standard token claim; unix date
	TokenString string    `json:"token_string"`
}

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
