package auth

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/database"
)

func InitializeGormModels() *gorm.DB {
	db := database.GetGORMDbConnection()

	defer db.Close()

	// Migrate the schema
	// Add your new models here
	db.AutoMigrate(
		&ACO{},
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

func (token *Token) BeforeSave() error {
	backend := InitAuthBackend()
	// Parse the value into a token.  If this works, it needs to be hashed before saving
	jwtToken, err := backend.GetJWTToken(token.Value)
	// If the parse to jwtToken fails then the value is already hashed (or not valid for other reasons) and no need to rehash it
	if err != nil {
		return nil
	}
	hash := Hash{}
	// If the token is valid hash it. If not, mark it inactive and clear out the value
	if jwtToken.Valid {
		token.Value = hash.Generate(token.Value)
	} else {
		token.Value = hash.Generate("INVALID")
		token.Active = false
	}

	return nil
}

type User struct {
	gorm.Model
	UUID  uuid.UUID `gorm:"primary_key; type:char(36)" json:"uuid"` // uuid
	Name  string    `json:"name"`                                   // name
	Email string    `json:"email"`                                  // email
	Aco   ACO       `gorm:"foreignkey:AcoID;association_foreignkey:UUID"`
	AcoID uuid.UUID `gorm:"type:char(36)" json:"aco_id"` // aco_id
}
