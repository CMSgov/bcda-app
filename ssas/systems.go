package ssas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/auth/rsautils"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"
	"io"
	"io/ioutil"
	"log"
	"net/url"
)

const DEFAULT_SCOPE = "bcda-api"

func InitializeSystemModels() *gorm.DB {
	log.Println("Initialize system models")
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	db.AutoMigrate(
		&System{},
		&EncryptionKey{},
		&Secret{},
	)

	db.Model(&System{}).AddForeignKey("group_id", "groups(group_id)", "RESTRICT", "RESTRICT")
	db.Model(&EncryptionKey{}).AddForeignKey("system_id", "systems(id)", "RESTRICT", "RESTRICT")
	db.Model(&Secret{}).AddForeignKey("system_id", "systems(id)", "RESTRICT", "RESTRICT")

	return db
}

type System struct {
	gorm.Model
	GroupID        string          `json:"group_id"`
	ClientID       string          `json:"client_id"`
	SoftwareID     string          `json:"software_id"`
	ClientName     string          `json:"client_name"`
	ClientURI      string          `json:"client_uri"`
	APIScope       string          `json:"api_scope"`
	EncryptionKeys []EncryptionKey `json:"encryption_keys"`
	Secrets        []Secret        `json:"secrets"`
}

type EncryptionKey struct {
	gorm.Model
	Body     string `json:"body"`
	System   System `gorm:"foreignkey:SystemID;association_foreignkey:ID"`
	SystemID uint   `json:"system_id"`
}

type Secret struct {
	gorm.Model
	Hash     string `json:"hash"`
	System   System `gorm:"foreignkey:SystemID;association_foreignkey:ID"`
	SystemID uint   `json:"system_id"`
}

func (system *System) SaveSecret(hashedSecret string) error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	secret := Secret{
		Hash:     hashedSecret,
		SystemID: system.ID,
	}

	err := db.Where("system_id = ?", system.ID).Delete(&Secret{}).Error
	if err != nil {
		return fmt.Errorf("unable to soft delete previous secrets for clientID %s: %s", system.ClientID, err.Error())
	}

	err = db.Create(&secret).Error
	if err != nil {
		return fmt.Errorf("could not save secret for clientID %s: %s", system.ClientID, err.Error())
	}

	return nil
}

func (system *System) GetSecret() (string, error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	secret := Secret{}

	err := db.Where("system_id = ?", system.ID).First(&secret).Error
	if err != nil {
		return "", fmt.Errorf("unable to get hashed secret for clientID %s: %s", system.ClientID, err.Error())
	}

	if secret.Hash == "" {
		return "", fmt.Errorf("stored hash of secret for clientID %s is blank", system.ClientID)
	}

	return secret.Hash, nil
}

func (system *System) GetPublicKey() (*rsa.PublicKey, error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var encryptionKey EncryptionKey
	err := db.Where("system_id = ?", system.ID).Find(&encryptionKey).Error
	if err != nil {
		return nil, fmt.Errorf("cannot find public key for clientID %s: %s", system.ClientID, err.Error())
	}

	return rsautils.ReadPublicKey(encryptionKey.Body)
}

func (system *System) SavePublicKey(publicKey io.Reader) error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	k, err := ioutil.ReadAll(publicKey)
	if err != nil {
		return fmt.Errorf("cannot read public key for clientID %s: %s", system.ClientID, err.Error())
	}

	key, err := rsautils.ReadPublicKey(string(k))
	if err != nil {
		return fmt.Errorf("invalid public key for clientID %s: %s", system.ClientID, err.Error())
	}
	if key == nil {
		return fmt.Errorf("invalid public key for clientID %s", system.ClientID)
	}

	encryptionKey := EncryptionKey{
		Body: string(k),
		SystemID: system.ID,
	}

	// Only one key should be valid per system.  Soft delete the currently active key, if any.
	err = db.Where("system_id = ?", system.ID).Delete(&EncryptionKey{}).Error
	if err != nil {
		return fmt.Errorf("unable to soft delete previous encryption keys for clientID %s: %s", system.ClientID, err.Error())
	}

	err = db.Create(&encryptionKey).Error
	if err != nil {
		return fmt.Errorf("could not save public key for clientID %s: %s", system.ClientID, err.Error())
	}

	return nil
}

// RevokeSystemKeyPair soft deletes the active encryption key
// for the specified system so that it can no longer be used
func (system *System) RevokeSystemKeyPair() error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var encryptionKey EncryptionKey

	err := db.Where("system_id = ?", system.ID).Find(&encryptionKey).Error
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
func (system *System) GenerateSystemKeyPair() (string, error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var key EncryptionKey
	if !db.Where("system_id = ?", system.ID).Find(&key).RecordNotFound() {
		return "", fmt.Errorf("encryption keypair already exists for system ID %d", system.ID)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("could not create key for system ID %d: %s", system.ID, err.Error())
	}

	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("could not marshal public key for system ID %d: %s", system.ID, err.Error())
	}

	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})

	encryptionKey := EncryptionKey{
		Body:     string(publicKeyBytes),
		SystemID: system.ID,
	}

	err = db.Create(&encryptionKey).Error
	if err != nil {
		return "", fmt.Errorf("could not save key for system ID %d: %s", system.ID, err.Error())
	}

	privateKeyBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)

	return string(privateKeyBytes), nil
}

func RegisterSystem(clientID string, clientName string, clientURI string, groupID string, scope string, publicKeyPEM string) (auth.Credentials, error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// A system is not valid without an active public key and a hashed secret.  However, they are stored separately in the
	// encryption_keys and secrets tables, requiring multiple INSERT statement.  To ensure we do not get into an invalid state,
	// wrap the two INSERT statements in a transaction.
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	creds := auth.Credentials{}

	regEvent := Event{Op: "RegisterSystem", TrackingID: clientID}
	OperationStarted(regEvent)

	if clientID == "" {
		regEvent.Help = "clientID is required"
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	if clientName == "" {
		regEvent.Help = "clientName is required"
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	if clientURI != "" && !isValidURI(clientURI) {
		regEvent.Help = "clientURI is invalid: " + clientURI
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	if scope == "" {
		scope = DEFAULT_SCOPE
	} else if scope != DEFAULT_SCOPE {
		regEvent.Help = "scope must be: " + DEFAULT_SCOPE
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	_, err := rsautils.ReadPublicKey(publicKeyPEM)
	if err != nil {
		regEvent.Help = "error in public key: " + err.Error()
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	system := System{
		GroupID:     groupID,
		ClientID: clientID,
		ClientName: clientName,
		ClientURI: clientURI,
		APIScope: scope,
	}

	err = tx.Create(&system).Error
	if err != nil {
		return creds, fmt.Errorf("could not save system for clientID %s, groupID %s: %s", clientID, groupID, err.Error())
	}

	encryptionKey := EncryptionKey{
		Body: publicKeyPEM,
		SystemID: system.ID,
	}

	// While the createEncryptionKey method below _could_ be called here (and system.SaveSecret() below),
	// we would lose the benefit of the transaction.
	err = tx.Create(&encryptionKey).Error
	if err != nil {
		return creds, fmt.Errorf("could not save public key for clientID %s, groupID %s: %s", clientID, groupID, err.Error())
	}

	clientSecret, err := GenerateSecret()
	if err != nil {
		regEvent.Help = fmt.Sprintf("cannot generate secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	hashedSecret, err := auth.NewHash(clientSecret)
	if err != nil {
		regEvent.Help = fmt.Sprintf("cannot generate hash of secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	secret := Secret{
		Hash: hashedSecret.String(),
		SystemID: system.ID,
	}

	err = tx.Create(&secret).Error
	if err != nil {
		regEvent.Help = fmt.Sprintf("cannot save secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	err = tx.Commit().Error
	if err != nil {
		regEvent.Help = fmt.Sprintf("could not commit transaction for new system with groupID %s: %s", groupID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	creds.ClientID = system.ClientID
	creds.ClientSecret = clientSecret
	creds.ClientName = system.ClientName

	OperationSucceeded(regEvent)
	return creds, nil
}

func GetSystemByClientID(clientID string) (System, error) {
	var (
		db = database.GetGORMDbConnection()
		system System
		err error
	)
	defer database.Close(db)

	if db.Find(&system, "client_id = ?", clientID).RecordNotFound() {
		err = errors.New("no System record found for " + clientID)
	}
	return system, err
}

// TODO: put this as a public function in the new plugin or in backend.go
func GenerateSecret() (string, error) {
	b := make([]byte, 40)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", b), nil
}

func isValidURI(u string) bool {
	_, err := url.ParseRequestURI(u)
	return err == nil
}