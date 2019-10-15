package ssas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
)

var DefaultScope string

const CredentialExpiration = 90 * 24 * time.Hour

func init() {
	getEnvVars()
}

func getEnvVars() {
	DefaultScope = os.Getenv("SSAS_DEFAULT_SYSTEM_SCOPE")

	if DefaultScope == "" {
		if os.Getenv("DEBUG") == "true" {
			DefaultScope = "bcda-api"
			return
		}
		ServiceHalted(Event{Help: "SSAS_DEFAULT_SYSTEM_SCOPE environment value must be set"})
		panic("SSAS_DEFAULT_SYSTEM_SCOPE environment value must be set")
	}
}

/*
	InitializeSystemModels will call gorm.DB.AutoMigrate() for models associated with systems, and set up foreign key
	relationships for those models if needed
*/
func InitializeSystemModels() *gorm.DB {
	log.Println("Initialize system models")
	db := GetGORMDbConnection()
	defer Close(db)

	db.Model(&System{}).AddForeignKey("group_id", "groups(group_id)", "RESTRICT", "RESTRICT")
	db.Model(&EncryptionKey{}).AddForeignKey("system_id", "systems(id)", "RESTRICT", "RESTRICT")
	db.Model(&Secret{}).AddForeignKey("system_id", "systems(id)", "RESTRICT", "RESTRICT")

	return db
}

type System struct {
	gorm.Model
	GroupID        string          `json:"group_id"`
	ClientID       string          `json:"client_id" gorm:"unique_index:idx_client"`
	SoftwareID     string          `json:"software_id"`
	ClientName     string          `json:"client_name"`
	APIScope       string          `json:"api_scope"`
	EncryptionKeys []EncryptionKey `json:"encryption_keys,omitempty"`
	Secrets        []Secret        `json:"secrets,omitempty"`
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

type AuthRegData struct {
	GroupID         string
	AllowedGroupIDs []string
	OktaID          string
}

/*
	SaveSecret should be provided with a secret hashed with ssas.NewHash(), which will
	be saved to the secrets table and associated with the current system.
*/
func (system *System) SaveSecret(hashedSecret string) error {
	db := GetGORMDbConnection()
	defer Close(db)

	secret := Secret{
		Hash:     hashedSecret,
		SystemID: system.ID,
	}

	if err := system.DeactivateSecrets(); err != nil {
		return err
	}

	if err := db.Create(&secret).Error; err != nil {
		return fmt.Errorf("could not save secret for clientID %s: %s", system.ClientID, err.Error())
	}
	SecretCreated(Event{Op: "SaveSecret", TrackingID: uuid.NewRandom().String(), ClientID: system.ClientID})

	return nil
}

/*
	GetSecret will retrieve the hashed secret associated with the current system.
*/
func (system *System) GetSecret() (string, error) {
	db := GetGORMDbConnection()
	defer Close(db)

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

/*
	DeactivateSecrets soft deletes secrets associated with the system.
*/
func (system *System) DeactivateSecrets() error {
	db := GetGORMDbConnection()
	defer Close(db)
	err := db.Where("system_id = ?", system.ID).Delete(&Secret{}).Error
	if err != nil {
		return fmt.Errorf("unable to soft delete previous secrets for clientID %s: %s", system.ClientID, err.Error())
	}
	return nil
}

/*
	GetEncryptionKey retrieves the key associated with the current system.
*/
func (system *System) GetEncryptionKey(trackingID string) (EncryptionKey, error) {
	db := GetGORMDbConnection()
	defer Close(db)

	getKeyEvent := Event{Op: "GetEncryptionKey", TrackingID: trackingID, ClientID: system.ClientID}
	OperationStarted(getKeyEvent)

	var encryptionKey EncryptionKey
	err := db.Where("system_id = ?", system.ID).Find(&encryptionKey).Error
	if err != nil {
		OperationFailed(getKeyEvent)
		return encryptionKey, fmt.Errorf("cannot find key for clientID %s: %s", system.ClientID, err.Error())
	}

	OperationSucceeded(getKeyEvent)
	return encryptionKey, nil
}

/*
	SavePublicKey should be provided with a public key in PEM format, which will be saved
	to the encryption_keys table and associated with the current system.
*/
func (system *System) SavePublicKey(publicKey io.Reader) error {
	db := GetGORMDbConnection()
	defer Close(db)

	k, err := ioutil.ReadAll(publicKey)
	if err != nil {
		return fmt.Errorf("cannot read public key for clientID %s: %s", system.ClientID, err.Error())
	}

	key, err := ReadPublicKey(string(k))
	if err != nil {
		return fmt.Errorf("invalid public key for clientID %s: %s", system.ClientID, err.Error())
	}
	if key == nil {
		return fmt.Errorf("invalid public key for clientID %s", system.ClientID)
	}

	encryptionKey := EncryptionKey{
		Body:     string(k),
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

/*
	RevokeSystemKeyPair soft deletes the active encryption key
	for the specified system so that it can no longer be used
*/
func (system *System) RevokeSystemKeyPair() error {
	db := GetGORMDbConnection()
	defer Close(db)

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
	db := GetGORMDbConnection()
	defer Close(db)

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

type Credentials struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	SystemID     string    `json:"system_id"`
	ClientName   string    `json:"client_name"`
	ExpiresAt    time.Time `json:"expires_at"`
}

/*
	RegisterSystem will save a new system and public key after verifying provided details for validity.  It returns
	a ssas.Credentials struct including the generated clientID and secret.
*/
func RegisterSystem(clientName string, groupID string, scope string, publicKeyPEM string, trackingID string) (Credentials, error) {
	db := GetGORMDbConnection()
	defer Close(db)

	// A system is not valid without an active public key and a hashed secret.  However, they are stored separately in the
	// encryption_keys and secrets tables, requiring multiple INSERT statement.  To ensure we do not get into an invalid state,
	// wrap the two INSERT statements in a transaction.
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	creds := Credentials{}
	clientID := uuid.NewRandom().String()

	// The caller of this function should have logged OperationCalled() with the same trackingID
	regEvent := Event{Op: "RegisterClient", TrackingID: trackingID, ClientID: clientID}
	OperationStarted(regEvent)

	if clientName == "" {
		regEvent.Help = "clientName is required"
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	if scope == "" {
		scope = DefaultScope
	} else if scope != DefaultScope {
		regEvent.Help = "scope must be: " + DefaultScope
		OperationFailed(regEvent)
		return creds, errors.New(regEvent.Help)
	}

	_, err := ReadPublicKey(publicKeyPEM)
	if err != nil {
		regEvent.Help = "error in public key: " + err.Error()
		OperationFailed(regEvent)
		return creds, errors.New("error in public key")
	}

	system := System{
		GroupID:    groupID,
		ClientID:   clientID,
		ClientName: clientName,
		APIScope:   scope,
	}

	err = tx.Create(&system).Error
	if err != nil {
		regEvent.Help = fmt.Sprintf("could not save system for clientID %s, groupID %s: %s", clientID, groupID, err.Error())
		OperationFailed(regEvent)
		// Returned errors are passed to API callers, and should include enough information to correct invalid submissions
		// without revealing implementation details.  CLI callers will be able to review logs for more information.
		return creds, errors.New("internal system error")
	}

	encryptionKey := EncryptionKey{
		Body:     publicKeyPEM,
		SystemID: system.ID,
	}

	// While the createEncryptionKey method below _could_ be called here (and system.SaveSecret() below),
	// we would lose the benefit of the transaction.
	err = tx.Create(&encryptionKey).Error
	if err != nil {
		regEvent.Help = fmt.Sprintf("could not save public key for clientID %s, groupID %s: %s", clientID, groupID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New("internal system error")
	}

	clientSecret, err := GenerateSecret()
	if err != nil {
		regEvent.Help = fmt.Sprintf("cannot generate secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New("internal system error")
	}

	hashedSecret, err := NewHash(clientSecret)
	if err != nil {
		regEvent.Help = fmt.Sprintf("cannot generate hash of secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New("internal system error")
	}

	secret := Secret{
		Hash:     hashedSecret.String(),
		SystemID: system.ID,
	}

	err = tx.Create(&secret).Error
	if err != nil {
		regEvent.Help = fmt.Sprintf("cannot save secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New("internal system error")
	}
	SecretCreated(regEvent)

	err = tx.Commit().Error
	if err != nil {
		regEvent.Help = fmt.Sprintf("could not commit transaction for new system with groupID %s: %s", groupID, err.Error())
		OperationFailed(regEvent)
		return creds, errors.New("internal system error")
	}

	creds.SystemID = fmt.Sprint(system.ID)
	creds.ClientID = system.ClientID
	creds.ClientSecret = clientSecret
	creds.ClientName = system.ClientName
	creds.ExpiresAt = time.Now().Add(CredentialExpiration)

	OperationSucceeded(regEvent)
	return creds, nil
}

// DataForSystem returns the group extra data associated with this system
func XDataFor(system System) (string, error) {
	group, err := GetGroupByGroupID(system.GroupID)
	if err != nil {
		return "", fmt.Errorf("no group for system %d; %s", system.ID, err)
	}
	Logger.Info("group xdata '", group, "'")
	// strconv.Unquote here?
	return group.XData, nil
}

//	GetSystemsByGroupID returns the systems associated with the provided group_id
func GetSystemsByGroupID(groupId string) ([]System, error) {
	var (
		db      = GetGORMDbConnection()
		systems []System
		err     error
	)
	defer Close(db)

	if err = db.Where("group_id = ?", groupId).Find(&systems).Error; err != nil {
		err = fmt.Errorf("no Systems found with group_id %s", groupId)
	}
	return systems, err
}

// GetSystemByClientID returns the system associated with the provided clientID
func GetSystemByClientID(clientID string) (System, error) {
	var (
		db     = GetGORMDbConnection()
		system System
		err    error
	)
	defer Close(db)

	if db.Find(&system, "client_id = ?", clientID).RecordNotFound() {
		err = fmt.Errorf("no System record found for client %s", clientID)
	}
	return system, err
}

// GetSystemByID returns the system associated with the provided ID
func GetSystemByID(id string) (System, error) {
	var (
		db     = GetGORMDbConnection()
		system System
		err    error
	)
	defer Close(db)

	if _, err = strconv.ParseUint(id, 10, 64); err != nil {
		return System{}, fmt.Errorf("invalid input %s; %s", id, err)
	}
	// must use the explicit where clause here because the id argument is a string
	if err = db.Find(&system, "id = ?", id).Error; err != nil {
		err = fmt.Errorf("no System record found with ID %s", id)
	}
	return system, err
}

func GenerateSecret() (string, error) {
	b := make([]byte, 40)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", b), nil
}

// ResetSecret creates a new secret for the current system.
func (system *System) ResetSecret(trackingID string) (Credentials, error) {
	db := GetGORMDbConnection()
	defer Close(db)

	creds := Credentials{}

	newSecretEvent := Event{Op: "ResetSecret", TrackingID: trackingID, ClientID: system.ClientID}
	OperationStarted(newSecretEvent)

	secretString, err := GenerateSecret()
	if err != nil {
		newSecretEvent.Help = fmt.Sprintf("could not reset secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(newSecretEvent)
		return creds, errors.New("internal system error")
	}

	hashedSecret, err := NewHash(secretString)
	if err != nil {
		newSecretEvent.Help = fmt.Sprintf("could not reset secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(newSecretEvent)
		return creds, errors.New("internal system error")
	}

	hashedSecretString := hashedSecret.String()
	if err = system.SaveSecret(hashedSecretString); err != nil {
		newSecretEvent.Help = fmt.Sprintf("could not reset secret for clientID %s: %s", system.ClientID, err.Error())
		OperationFailed(newSecretEvent)
		return creds, errors.New("internal system error")
	}

	OperationSucceeded(newSecretEvent)

	creds.ClientID = system.ClientID
	creds.ClientSecret = secretString
	creds.ClientName = system.ClientName
	creds.ExpiresAt = time.Now().Add(CredentialExpiration)
	return creds, nil
}

// CleanDatabase deletes the given group and associated systems, encryption keys, and secrets.
func CleanDatabase(group Group) error {
	var (
		system        System
		encryptionKey EncryptionKey
		secret        Secret
		systemIds     []int
		db            = GetGORMDbConnection()
	)
	defer Close(db)

	if group.ID == 0 {
		return fmt.Errorf("invalid group.ID")
	}

	foundGroup := Group{GroupID: group.GroupID}
	err := db.Unscoped().Find(&foundGroup).Error
	if err != nil {
		return fmt.Errorf("unable to find group %s: %s", group.GroupID, err.Error())
	}

	err = db.Table("systems").Where("group_id = ?", group.GroupID).Pluck("ID", &systemIds).Error
	if err != nil {
		Logger.Errorf("unable to find associated systems: %s", err.Error())
	} else {
		err = db.Unscoped().Where("system_id IN (?)", systemIds).Delete(&encryptionKey).Error
		if err != nil {
			Logger.Errorf("unable to delete encryption keys: %s", err.Error())
		}

		err = db.Unscoped().Where("system_id IN (?)", systemIds).Delete(&secret).Error
		if err != nil {
			Logger.Errorf("unable to delete secrets: %s", err.Error())
		}

		err = db.Unscoped().Where("id IN (?)", systemIds).Delete(&system).Error
		if err != nil {
			Logger.Errorf("unable to delete systems: %s", err.Error())
		}
	}

	err = db.Unscoped().Delete(&group).Error
	if err != nil {
		return fmt.Errorf("unable to delete group: %s", err.Error())
	}

	return nil
}
