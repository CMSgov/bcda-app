package auth_test

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
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
	database.Close(s.db)
}

func (s *ModelsTestSuite) TestTokenCreation() {
	tokenUUID := uuid.NewRandom()
	acoUUID := uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	userUUID := uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	issuedAt := time.Now().Unix()
	expiresOn := time.Now().Add(time.Hour * time.Duration(72)).Unix()

	tokenString, err := auth.GenerateTokenString(
		tokenUUID.String(),
		userUUID.String(),
		acoUUID.String(),
		issuedAt,
		expiresOn,
	)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tokenString)

	// Get the claims of the token to find the token ID that was created
	token := auth.Token{
		UUID:      tokenUUID,
		UserID:    userUUID,
		Value:     tokenString,
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

func (s *BackendTestSuite) TestGenerateTokenString() {
	var (
		userUUID = "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
		acoUUID  = "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	)
	token, err := auth.TokenStringWithIDs(uuid.NewRandom().String(), userUUID, acoUUID)

	// No errors, token is not nil
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Wipe the keys
	s.AuthBackend.PrivateKey = nil
	s.AuthBackend.PublicKey = nil
	defer s.AuthBackend.ResetAlphaBackend()
	assert.Panics(s.T(), func() {
		_, _ = auth.TokenStringWithIDs(uuid.NewRandom().String(), userUUID, acoUUID)
	})
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
