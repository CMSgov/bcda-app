package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
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
