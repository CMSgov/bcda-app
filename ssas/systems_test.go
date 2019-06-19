package ssas

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SystemsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *SystemsTestSuite) SetupSuite() {
	s.db = database.GetGORMDbConnection()
	InitializeGroupModels()
	InitializeSystemModels()
}

func (s *SystemsTestSuite) TearDownSuite() {
	database.Close(s.db)
}

func (s *SystemsTestSuite) AfterTest() {
}

func (s *SystemsTestSuite) TestRevokeSystemKeyPair() {
	assert := s.Assert()

	group := Group{GroupID: "A00001"}
	s.db.Save(&group)
	system := System{GroupID: group.GroupID}
	s.db.Save(&system)
	encryptionKey := EncryptionKey{SystemID: system.ID}
	s.db.Save(&encryptionKey)

	err := system.RevokeSystemKeyPair()
	assert.Nil(err)
	assert.Empty(system.EncryptionKeys)
	s.db.Unscoped().Find(&encryptionKey)
	assert.NotNil(encryptionKey.DeletedAt)

	s.db.Unscoped().Delete(&encryptionKey)
	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func (s *SystemsTestSuite) TestGenerateSystemKeyPair() {
	assert := s.Assert()

	group := Group{GroupID: "abcdef123456"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	privateKeyStr, err := system.GenerateSystemKeyPair()
	assert.NoError(err)
	assert.NotEmpty(privateKeyStr)

	privKeyBlock, _ := pem.Decode([]byte(privateKeyStr))
	if privKeyBlock == nil {
		s.FailNow("unable to decode private key ", privateKeyStr)
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privKeyBlock.Bytes)
	if err != nil {
		s.FailNow(err.Error())
	}

	var pubEncrKey EncryptionKey
	err = s.db.First(&pubEncrKey, "system_id = ?", system.ID).Error
	if err != nil {
		s.FailNow(err.Error())
	}
	pubKeyBlock, _ := pem.Decode([]byte(pubEncrKey.Body))
	publicKey, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
	if err != nil {
		s.FailNow(err.Error())
	}
	assert.Equal(&privateKey.PublicKey, publicKey)

	s.db.Unscoped().Delete(&pubEncrKey)
	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func (s *SystemsTestSuite) TestGenerateSystemKeyPair_AlreadyExists() {
	assert := s.Assert()

	group := Group{GroupID: "bcdefa234561"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	encrKey := EncryptionKey{
		SystemID: system.ID,
	}
	err = s.db.Create(&encrKey).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	privateKey, err := system.GenerateSystemKeyPair()
	systemIDStr := strconv.FormatUint(uint64(system.ID), 10)
	assert.EqualError(err, "encryption keypair already exists for system ID "+systemIDStr)
	assert.Empty(privateKey)

	s.db.Unscoped().Delete(&encrKey)
	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func (s *SystemsTestSuite) TestEncryptionKeyModel() {
	assert := s.Assert()

	group := Group{GroupID: "A00000"}
	s.db.Save(&group)

	system := System{GroupID: "A00000"}
	s.db.Save(&system)

	systemIDStr := strconv.FormatUint(uint64(system.ID), 10)
	encryptionKeyBytes := []byte(`{"body": "this is a public key", "system_id": ` + systemIDStr + `}`)
	encryptionKey := EncryptionKey{}
	err := json.Unmarshal(encryptionKeyBytes, &encryptionKey)
	assert.Nil(err)

	err = s.db.Save(&encryptionKey).Error
	assert.Nil(err)

	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func TestSystemsTestSuite(t *testing.T) {
	suite.Run(t, new(SystemsTestSuite))
}