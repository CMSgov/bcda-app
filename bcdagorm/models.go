package bcdagorm

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/database"
)

func Initialize() *gorm.DB {
	db := database.GetGORMDbConnection()

	defer db.Close()

	// Migrate the schema
	// Add your new models here
	db.AutoMigrate(
		&ACO{},
		&Job{},
		&Token{},
		&User{},
	)

	return db
}

type ACO struct {
	gorm.Model
	UUID uuid.UUID `gorm:"primary_key; type:char(36)" json:"uuid"` // uuid
	Name string    `json:"name"`                                   // name
}

type Job struct {
	gorm.Model
	Aco        ACO       `gorm:"foreignkey:AcoID;association_foreignkey:UUID"` // aco
	AcoID      uuid.UUID `gorm:"primary_key; type:char(36)" json:"aco_id"`
	User       User      `gorm:"foreignkey:UserID;association_foreignkey:UUID"` // user
	UserID     uuid.UUID `gorm:"type:char(36)"`
	RequestURL string    `json:"request_url"` // request_url
	Status     string    `json:"status"`      // status
}

type Token struct {
	gorm.Model
	UUID   uuid.UUID `gorm:"primary_key" json:"uuid"` // uuid
	User   User      `gorm:"foreignkey:UserID;association_foreignkey:UUID"`
	UserID uuid.UUID `json:"user_id"`                                // user_id
	Value  string    `gorm:"type:varchar(511); unique" json:"value"` // value
	Active bool      `json:"active"`                                 // active

}

type User struct {
	gorm.Model
	UUID  uuid.UUID `gorm:"primary_key; type:char(36)" json:"uuid"` // uuid
	Name  string    `json:"name"`                                   // name
	Email string    `json:"email"`                                  // email
	Aco   ACO       `gorm:"foreignkey:AcoID;association_foreignkey:UUID"`
	AcoID uuid.UUID `gorm:"type:char(36)" json:"aco_id"` // aco_id
}
