package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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
	db.AutoMigrate(
		&Token{},
	)

	// force manual deletion of foreign key and this related record (you can delete a Token, but not an aco with a token
	db.Model(&Token{}).AddForeignKey("aco_id", "acos(uuid)", "RESTRICT", "RESTRICT")

	return db
}

type Token struct {
	gorm.Model
	// even though gorm.Model has an `id` field declared as the primary key, the following definition overrides that
	UUID        uuid.UUID  `gorm:"primary_key" json:"uuid"`                      // uuid (primary key)
	Value       string     `gorm:"type:varchar(511); unique" json:"value"`       // Deprecated: When can we drop Value without hurting existing alpha tokens?
	Active      bool       `json:"active"`                                       // active
	ACO         models.ACO `gorm:"foreignkey:ACOID;association_foreignkey:UUID"` // ACO needed here because user can belong to multiple ACOs
	ACOID       uuid.UUID  `gorm:"type:uuid" json:"aco_id"`
	IssuedAt    int64      `json:"issued_at"`  // standard token claim; unix date
	ExpiresOn   int64      `json:"expires_on"` // standard token claim; unix date
	TokenString string     `gorm:"-"`          // ignore; not for database
}

// When getting a Token out of the database, reconstruct its string value and store it in TokenString.
func (t *Token) AfterFind() error {
	s, err := GenerateTokenString(t.UUID.String(), t.ACOID.String(), t.IssuedAt, t.ExpiresOn)
	if err == nil {
		t.TokenString = s
		return nil
	}
	return err
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

// RevokeSystemKeyPair soft deletes the active encryption key
// for the specified system so that it can no longer be used
func RevokeSystemKeyPair(systemID uint) error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	var system models.System

	err := db.Preload("EncryptionKeys").Find(&system, systemID).Error
	if err != nil {
		return err
	}

	var encryptionKey models.EncryptionKey
	err = db.Find(&encryptionKey, system.EncryptionKeys[0].ID).Error
	if err != nil {
		return err
	}

	err = db.Delete(&encryptionKey).Error
	if err != nil {
		return err
	}

	return nil
}

/*
 GenerateSystemKeyPair creates a keypair for a system. The public key is saved to the database and the private key is returned.
*/
func GenerateSystemKeyPair(systemID uint) (string, error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var system models.System
	err := db.Preload("EncryptionKeys").First(&system, systemID).Error
	if err != nil {
		return "", errors.Wrapf(err, "could not find system ID %d", systemID)
	}

	if len(system.EncryptionKeys) > 0 {
		return "", fmt.Errorf("encryption keypair already exists for system ID %d", systemID)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", errors.Wrapf(err, "could not create key for system ID %d", systemID)
	}

	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal public key for system ID %d", systemID)
	}

	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})

	encryptionKey := models.EncryptionKey{
		Body:     string(publicKeyBytes),
		SystemID: system.ID,
	}

	err = db.Create(&encryptionKey).Error
	if err != nil {
		return "", errors.Wrapf(err, "could not save key for system ID %d", systemID)
	}

	privateKeyBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)

	return string(privateKeyBytes), nil
}
func CreateAlphaToken(ttl int, acoSize string) (string, error) {
	aco, err := createAlphaEntities(acoSize)
	if err != nil {
		return "", err
	}

	creds, err := GetProvider().RegisterClient(aco.UUID.String())
	if err != nil {
		return "", fmt.Errorf("could not register client for %s (%s) because %s", aco.UUID.String(), aco.Name, err.Error())
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	// Only update aco.ClientID.  Other attributes of this ACO (AlphaSecret) may have been altered in the database by the
	// RegisterClient() call above, so we should not save these potentially stale values.
	err = db.Model(&aco).Update("client_id", creds.ClientID).Error
	if err != nil {
		return "", fmt.Errorf("could not save ClientID %s to ACO %s (%s) because %s", aco.ClientID, aco.UUID.String(), aco.Name, err.Error())
	}

	msg := fmt.Sprintf("%s\n%s\n%s", creds.ClientName, creds.ClientID, creds.ClientSecret)

	return msg, nil
}

func createAlphaEntities(acoSize string) (aco models.ACO, err error) {
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

	aco, err = models.CreateAlphaACO(tx)
	if err != nil {
		tx.Rollback()
		return aco, err
	}

	if err = models.AssignAlphaBeneficiaries(tx, aco, acoSize); err != nil {
		tx.Rollback()
		return aco, err
	}

	if tx.Commit().Error != nil {
		tx.Rollback()
		return aco, tx.Error
	}

	return aco, nil
}
