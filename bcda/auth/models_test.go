package auth_test

import (
	"crypto/x509"
	"encoding/pem"
	"strconv"
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
	group := models.Group{GroupID: "A00001"}
	s.db.Save(&group)
	defer s.db.Unscoped().Delete(&group)
	system := models.System{GroupID: group.GroupID}
	s.db.Save(&system)
	defer s.db.Unscoped().Delete(&system)
	encryptionKey := models.EncryptionKey{SystemID: system.ID}
	s.db.Save(&encryptionKey)
	defer s.db.Unscoped().Delete(&encryptionKey)

	err := auth.RevokeSystemKeyPair(system.ID)
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), system.EncryptionKeys)
	s.db.Unscoped().Find(&encryptionKey)
	assert.NotNil(s.T(), encryptionKey.DeletedAt)
}

func (s *ModelsTestSuite) TestGenerateSystemKeyPair() {
	group := models.Group{GroupID: "abcdef123456"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := models.System{GroupID: group.GroupID}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	privateKeyStr, err := auth.GenerateSystemKeyPair(system.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), privateKeyStr)

	privKeyBlock, _ := pem.Decode([]byte(privateKeyStr))
	privateKey, err := x509.ParsePKCS1PrivateKey(privKeyBlock.Bytes)
	if err != nil {
		s.FailNow(err.Error())
	}

	var pubEncrKey models.EncryptionKey
	err = s.db.First(&pubEncrKey, "system_id = ?", system.ID).Error
	if err != nil {
		s.FailNow(err.Error())
	}
	pubKeyBlock, _ := pem.Decode([]byte(pubEncrKey.Body))
	publicKey, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
	if err != nil {
		s.FailNow(err.Error())
	}
	assert.Equal(s.T(), &privateKey.PublicKey, publicKey)

	s.db.Unscoped().Delete(&pubEncrKey)
	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func (s *ModelsTestSuite) TestGenerateSystemKeyPair_AlreadyExists() {
	group := models.Group{GroupID: "bcdefa234561"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := models.System{GroupID: group.GroupID}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	encrKey := models.EncryptionKey{
		SystemID: system.ID,
	}
	err = s.db.Create(&encrKey).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	privateKey, err := auth.GenerateSystemKeyPair(system.ID)
	systemIDStr := strconv.FormatUint(uint64(system.ID), 10)
	assert.EqualError(s.T(), err, "encryption keypair already exists for system ID "+systemIDStr)
	assert.Empty(s.T(), privateKey)

	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
