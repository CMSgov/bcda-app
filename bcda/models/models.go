package models

import (
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
)

func InitializeGormModels() *gorm.DB {
	db := database.GetGORMDbConnection()
	defer db.Close()

	// Migrate the schema
	// Add your new models here
	db.AutoMigrate(
		&Job{},
		&JobKey{},
	)

	return db
}

type Job struct {
	gorm.Model
	Aco        auth.ACO  `gorm:"foreignkey:AcoID;association_foreignkey:UUID"` // aco
	AcoID      uuid.UUID `gorm:"primary_key; type:char(36)" json:"aco_id"`
	User       auth.User `gorm:"foreignkey:UserID;association_foreignkey:UUID"` // user
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
