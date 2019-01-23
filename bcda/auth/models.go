package auth

import (
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pborman/uuid"

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

	return db
}

type Token struct {
	gorm.Model
	// even though gorm.Model has an `id` field declared as the primary key, the following definition overrides that
	UUID        uuid.UUID   `gorm:"primary_key" json:"uuid"` // uuid (primary key)
	User        models.User `gorm:"foreignkey:UserID;association_foreignkey:UUID"`
	UserID      uuid.UUID   `json:"user_id"`                                // user_id
	Value       string      `gorm:"type:varchar(511); unique" json:"value"` // value
	Active      bool        `json:"active"`                                 // active
	Aco         models.ACO  `gorm:"foreignkey:AcoID;association_foreignkey:UUID"`
	AcoID       uuid.UUID   `json:"aco_id"`     // aco_id
	IssuedAt    int64       `json:"issued_at"`  // standard token claim; unix date
	ExpiresOn   int64       `json:"expires_on"` // standard token claim; unix date
	Token       jwt.Token   `gorm:"-"`          // ignore; not for database
	TokenString string      `gorm:"-"`          // ignore; not for database
	// we store AcoID on the token because a user can belong to multiple ACOs
	// unix time converter here: http://unixepoch.com
}

// When getting a Token out of the DB, reconstruct its tokenString
func (t *Token) AfterFind() error {
	s, err := GenerateTokenString(t.UUID, t.UserID, t.AcoID, t.IssuedAt, t.ExpiresOn)
	if err == nil {
		t.TokenString = s
		return nil
	}
	return err
}

// func (t *Token) BeforeSave() error {
// 	backend := InitAuthBackend()
// 	// Parse the value into a t.  If this method does not return an error, the t needs to be hashed before saving
// 	jwtToken, err := backend.GetJWToken(t.Value)
// 	// If we got an error, the t is already hashed (or not valid for other reasons) and no need to rehash it
// 	if err != nil {
// 		return nil
// 	}
// 	hash := Hash{}
// 	// If the token is valid hash it. If not, mark it inactive and clear out the value
// 	// Why do we do this? all the data in the token is in the db already, so there's no data that is precious
// 	// moreover, it may not be possible to reconstruct the token string if we need it
// 	if jwtToken.Valid {
// 		t.Value = hash.Generate(t.Value)
// 	} else {
// 		t.Value = hash.Generate("INVALID")
// 		t.Active = false
// 	}
//
// 	return nil
// }

// Given all claim values, construct a token string. This is mostly a helper method for testing (we think)
func GenerateTokenString(id, userID, acoID uuid.UUID, issuedAt int64, expiresOn int64) (string, error) {
	token := jwt.New(jwt.SigningMethodRS512)
	token.Claims = jwt.MapClaims{
		"exp": expiresOn,
		"iat": issuedAt,
		"sub": userID.String(),
		"aco": acoID.String(),
		"id":  id.String(),
	}
	return token.SignedString(InitAuthBackend().PrivateKey)
}
