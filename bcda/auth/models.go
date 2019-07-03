package auth

import (
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
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

func GetACOByClientID(clientID string) (models.ACO, error) {
	var (
		db  = database.GetGORMDbConnection()
		aco models.ACO
		err error
	)
	defer database.Close(db)

	if db.Find(&aco, "client_id = ?", clientID).RecordNotFound() {
		err = errors.New("no ACO record found for " + clientID)
	}
	return aco, err
}

func GetACOByCMSID(cmsID string) (models.ACO, error) {
	var (
		db  = database.GetGORMDbConnection()
		aco models.ACO
		err error
	)
	defer database.Close(db)

	if db.Find(&aco, "cms_id = ?", cmsID).RecordNotFound() {
		err = errors.New("no ACO record found for " + cmsID)
	}
	return aco, err
}

func CreateAlphaToken(ttl int, acoCMSID string) (string, error) {
	aco, err := createAlphaEntities(acoCMSID)
	if err != nil {
		return "", err
	}

	creds, err := GetProvider().RegisterSystem(aco.UUID.String())
	if err != nil {
		return "", fmt.Errorf("could not register client for %s (%s) because %s", aco.UUID.String(), aco.Name, err.Error())
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	// Only update aco.ClientID.  Other attributes of this ACO (AlphaSecret) may have been altered in the database by the
	// RegisterSystem() call above, so we should not save these potentially stale values.
	err = db.Model(&aco).Update("client_id", creds.ClientID).Error
	if err != nil {
		return "", fmt.Errorf("could not save ClientID %s to ACO %s (%s) because %s", aco.ClientID, aco.UUID.String(), aco.Name, err.Error())
	}

	msg := fmt.Sprintf("%s\n%s\n%s", creds.ClientName, creds.ClientID, creds.ClientSecret)

	return msg, nil
}

func createAlphaEntities(acoCMSID string) (aco models.ACO, err error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			if tx.Error != nil {
				tx.Rollback()
			}
			err = fmt.Errorf("createAlphaEntities failed because %s", r)
		}
	}()

	if tx.Error != nil {
		return aco, tx.Error
	}

	aco, err = models.CreateAlphaACO(acoCMSID, tx)
	if err != nil {
		tx.Rollback()
		return aco, err
	}

	if tx.Commit().Error != nil {
		tx.Rollback()
		return aco, tx.Error
	}

	return aco, nil
}
