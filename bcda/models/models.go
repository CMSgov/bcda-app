package models

import (
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/secutils"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
)

func InitializeGormModels() *gorm.DB {
	fmt.Print("Initialize bcda models")
	db := database.GetGORMDbConnection()
	defer db.Close()

	// Migrate the schema
	// Add your new models here
	// This should probably not be called in production
	// What happens when you need to make a database change, there's already data you need to preserve, and
	// you need to run a script to migrate existing data to its new home or shape?
	db.AutoMigrate(
		&ACO{},
		&User{},
		&Job{},
		&JobKey{},
	)

	return db
}

type Job struct {
	gorm.Model
	Aco        ACO       `gorm:"foreignkey:AcoID;association_foreignkey:UUID"` // aco
	AcoID      uuid.UUID `gorm:"primary_key; type:char(36)" json:"aco_id"`
	User       User      `gorm:"foreignkey:UserID;association_foreignkey:UUID"` // user
	UserID     uuid.UUID `gorm:"type:char(36)"`
	RequestURL string    `json:"request_url"` // request_url
	Status     string    `json:"status"`      // status
	JobKeys    []JobKey
}

type JobKey struct {
	gorm.Model
	Job          Job  `gorm:"foreignkey:jobID"`
	JobID        uint `gorm:"primary_key" json:"job_id"`
	EncryptedKey []byte
	FileName     string `gorm:"type:char(127)"`
}

type ACO struct {
	gorm.Model
	UUID     uuid.UUID `gorm:"primary_key; type:char(36)" json:"uuid"` // uuid
	Name     string    `json:"name"`                                   // name
	ClientID string    `json:"client_id"`                              // software client id
}

func (aco *ACO) GetPublicKey() *rsa.PublicKey {
	// todo implement a real thing.  But for now we can use this.
	return GetATOPublicKey()
}

// This exists to provide a known static keys used for ACO's in our alpha tests.
// This key is not meant to protect anything and both halves will be made available publicly
func GetATOPublicKey() *rsa.PublicKey {
	fmt.Println("Looking for a key at:")
	fmt.Println(os.Getenv("ATO_PUBLIC_KEY_FILE"))
	atoPublicKeyFile, err := os.Open(os.Getenv("ATO_PUBLIC_KEY_FILE"))
	if err != nil {
		fmt.Println("failed to open file")
		panic(err)
	}
	return secutils.OpenPublicKeyFile(atoPublicKeyFile)
}

func GetATOPrivateKey() *rsa.PrivateKey {
	atoPrivateKeyFile, err := os.Open(os.Getenv("ATO_PRIVATE_KEY_FILE"))
	if err != nil {
		panic(err)
	}
	return secutils.OpenPrivateKeyFile(atoPrivateKeyFile)
}

func CreateACO(name string) (uuid.UUID, error) {
	db := database.GetGORMDbConnection()
	defer db.Close()

	aco := ACO{Name: name, UUID: uuid.NewRandom()}
	db.Create(&aco)

	return aco.UUID, db.Error
}

type User struct {
	gorm.Model
	UUID  uuid.UUID `gorm:"primary_key; type:char(36)" json:"uuid"` // uuid
	Name  string    `json:"name"`                                   // name
	Email string    `json:"email"`                                  // email
	Aco   ACO       `gorm:"foreignkey:AcoID;association_foreignkey:UUID"`
	AcoID uuid.UUID `gorm:"type:char(36)" json:"aco_id"` // aco_id
}

func CreateUser(name string, email string, acoUUID uuid.UUID) (User, error) {
	db := database.GetGORMDbConnection()
	defer db.Close()
	var aco ACO
	var user User
	// If we don't find the ACO return a blank user and an error
	if db.First(&aco, "UUID = ?", acoUUID).RecordNotFound() {
		return user, fmt.Errorf("unable to locate ACO with id of %v", acoUUID)
	}
	// check for duplicate email addresses and only make one if it isn't found
	if db.First(&user, "email = ?", email).RecordNotFound() {
		user = User{UUID: uuid.NewRandom(), Name: name, Email: email, AcoID: aco.UUID}
		db.Create(&user)
		return user, nil
	} else {
		return user, fmt.Errorf("unable to create user for %v because a user with that Email address already exists", email)
	}
}

// CLI command only support; note that we are choosing to fail quickly and let the user (one of us) figure it out
func CreateAlphaACO(db *gorm.DB) (ACO, error) {
	var count int
	db.Table("acos").Count(&count)
	aco := ACO{Name: fmt.Sprintf("Alpha ACO %d", count), UUID: uuid.NewRandom()}
	db.Create(&aco)

	return aco, db.Error
}

func AssignAlphaBeneficiaries(db *gorm.DB, aco ACO, acoSize string) error {
	s := "insert into beneficiaries (patient_id, aco_id) select patient_id, '" + aco.UUID.String() +
		"' from beneficiaries where aco_id = (select uuid from acos where name ilike 'ACO " + acoSize + "')"
	return db.Exec(s).Error
}

// CLI command only support; note that we are choosing to fail quickly and let the user (one of us) figure it out
func CreateAlphaUser(db *gorm.DB, aco ACO) (User, error) {
	var count int
	db.Table("users").Count(&count)
	user := User{UUID: uuid.NewRandom(),
		Name:  fmt.Sprintf("Alpha User%d", count),
		Email: fmt.Sprintf("alpha.user.%d@nosuchdomain.com", count), AcoID: aco.UUID}
	db.Create(&user)

	return user, db.Error
}
