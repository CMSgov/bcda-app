package auth_test

import (
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupSuite() {
	testUtils.SetUnitTestKeysForAuth()
}

func (s *ModelsTestSuite) SetupTest() {
	s.db = database.GetGORMDbConnection()
}

func (s *ModelsTestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *ModelsTestSuite) TestTokenCreation() {
	tokenUUID := uuid.NewRandom()
	acoUUID := uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	issuedAt := time.Now().Unix()
	expiresOn := time.Now().Add(time.Hour * time.Duration(72)).Unix()

	tokenString, err := auth.GenerateTokenString(
		tokenUUID.String(),
		acoUUID.String(),
		issuedAt,
		expiresOn,
	)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tokenString)

	// Get the claims of the token to find the token ID that was created
	token := auth.Token{
		UUID:      tokenUUID,
		Active:    true,
		ACOID:     acoUUID,
		IssuedAt:  issuedAt,
		ExpiresOn: expiresOn,
	}
	s.db.Create(&token)

	var savedToken auth.Token
	s.db.Find(&savedToken, "UUID = ?", tokenUUID)
	assert.NotNil(s.T(), savedToken)
	assert.Equal(s.T(), tokenString, savedToken.TokenString)
}

func (s *ModelsTestSuite) TestRevokeSystemKeyPair() {
	encryptionKey := models.EncryptionKey{SystemID: 1}
	s.db.Save(&models.Group{GroupID: "A00001"})
	s.db.Save(&models.System{GroupID: "A00001"})
	s.db.Save(&encryptionKey)
	encryptionKey, err := auth.RevokeSystemKeyPair(encryptionKey.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), encryptionKey.DeletedAt)
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
