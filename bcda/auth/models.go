package auth

import (
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"
)

func InitializeGormModels() *gorm.DB {
	db := database.GetGORMDbConnection()
	defer db.Close()

	// Migrate the schema
	// Add your new models here
	db.AutoMigrate(
		&Token{},
	)

	return db
}

type Token struct {
	gorm.Model
	UUID   uuid.UUID   `gorm:"primary_key" json:"uuid"` // uuid
	User   models.User `gorm:"foreignkey:UserID;association_foreignkey:UUID"`
	UserID uuid.UUID   `json:"user_id"`                                // user_id
	Value  string      `gorm:"type:varchar(511); unique" json:"value"` // value
	Active bool        `json:"active"`                                 // active
	Token jwt.Token    `gorm:"-"`									   // ignore; not for database
}

func (token *Token) BeforeSave() error {
	backend := InitAuthBackend()
	// Parse the value into a token.  If this works, it needs to be hashed before saving
	jwtToken, err := backend.GetJWToken(token.Value)
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
