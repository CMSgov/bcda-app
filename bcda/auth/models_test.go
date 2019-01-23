package auth_test

import (
	"testing"

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
	testUtils.AuthTestSuite
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupTest() {
	// Setup the DB
	auth.InitializeGormModels()
	s.db = database.GetGORMDbConnection()
	s.SetupAuthBackend()
}

func (s *ModelsTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *ModelsTestSuite) TestTokenCreation() {
	acoUUID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	userUUID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"

	tokenString, err := s.AuthBackend.GenerateTokenString(
		userUUID,
		acoUUID,
	)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tokenString)

	var user models.User
	s.db.Find(&user, "UUID = ?", userUUID)

	// Get the claims of the token to find the token ID that was created
	claims := s.AuthBackend.GetJWTClaims(tokenString)
	tokenUUID := claims["id"].(string)
	token := auth.Token{
		UUID:   uuid.Parse(tokenUUID),
		UserID: user.UUID,
		Value:  tokenString,
		Active: true,
	}
	s.db.Create(&token)

	var savedToken auth.Token
	s.db.Find(&savedToken, "UUID = ?", tokenUUID)
	assert.NotNil(s.T(), savedToken)
	assert.Equal(s.T(), token.Value, savedToken.Value)
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
