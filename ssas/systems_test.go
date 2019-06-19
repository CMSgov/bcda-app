package ssas

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"

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

	err = s.cleanDatabase(group)
	assert.Nil(err)
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

	err = s.cleanDatabase(group)
	assert.Nil(err)
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

	err = s.cleanDatabase(group)
	assert.Nil(err)
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

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestSaveSecret() {
	assert := s.Assert()

	group := Group{GroupID: "T21212"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	// First secret should save
	secret1, err := generateSecret()
	if err != nil {
		s.FailNow("cannot generate random secret")
	}
	hashedSecret1, err := auth.NewHash(secret1)
	if err != nil {
		s.FailNow("cannot hash random secret")
	}
	err = system.SaveSecret(hashedSecret1.String())
	if err != nil {
		s.FailNow(err.Error())
	}

	// Second secret should cause first secret to be soft-deleted
	secret2, err := generateSecret()
	if err != nil {
		s.FailNow("cannot generate random secret")
	}
	hashedSecret2, err := auth.NewHash(secret2)
	if err != nil {
		s.FailNow("cannot hash random secret")
	}
	err = system.SaveSecret(hashedSecret2.String())
	if err != nil {
		s.FailNow(err.Error())
	}

	// Verify we now retrieve second secret
	// Note that this also tests GetSecret()
	savedHash, err := system.GetSecret()
	if err != nil {
		s.FailNow(err.Error())
	}
	assert.True(auth.Hash(savedHash).IsHashOf(secret2))

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) cleanDatabase(group Group) error {
	var system System
	var encryptionKey EncryptionKey
	var secret Secret
	var systemIds []int
	err := s.db.Where("group_id = ?", group.GroupID).Find(&system).Pluck("id", &systemIds).Error
	if err != nil {
		return fmt.Errorf("unable to find associated systems: %s", err.Error())
	}

	err = s.db.Unscoped().Where("system_id IN (?)", systemIds).Delete(&encryptionKey).Error
	if err != nil {
		return fmt.Errorf("unable to delete encryption keys: %s", err.Error())
	}

	err = s.db.Unscoped().Where("system_id IN (?)", systemIds).Delete(&secret).Error
	if err != nil {
		return fmt.Errorf("unable to delete secrets: %s", err.Error())
	}

	err = s.db.Unscoped().Where("id IN (?)", systemIds).Delete(&system).Error
	if err != nil {
		return fmt.Errorf("unable to delete systems: %s", err.Error())
	}

	err = s.db.Unscoped().Delete(&group).Error
	if err != nil {
		return fmt.Errorf("unable to delete group: %s", err.Error())
	}

	return nil
}

// TODO: put this as a public function in the new plugin or in backend.go
func generateSecret() (string, error) {
	b := make([]byte, 40)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", b), nil
}

func TestSystemsTestSuite(t *testing.T) {
	suite.Run(t, new(SystemsTestSuite))
}
