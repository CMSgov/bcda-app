package ssas

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type SystemsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *SystemsTestSuite) SetupSuite() {
	s.db = GetGORMDbConnection()
	InitializeGroupModels()
	InitializeSystemModels()
}

func (s *SystemsTestSuite) TearDownSuite() {
	Close(s.db)
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

func (s *SystemsTestSuite) TestSystemSavePublicKey() {
	assert := s.Assert()

	clientID := uuid.NewRandom().String()
	groupID := "T33333"

	// Setup Group and System
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	assert.Nil(err)
	system := System{ClientID: clientID, GroupID: groupID}
	err = s.db.Create(&system).Error
	assert.Nil(err)


	// Setup key
	keyPair, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(err, "error creating random test keypair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save key
	err = system.SavePublicKey(bytes.NewReader(publicKeyBytes))
	if err != nil {
		assert.FailNow("error saving key: " + err.Error())
	}

	// Retrieve and verify
	storedKey, err := system.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(storedKey)
	storedPublicKeyPKIX, err := x509.MarshalPKIXPublicKey(storedKey)
	assert.Nil(err, "unable to marshal saved public key")
	storedPublicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: storedPublicKeyPKIX,
	})
	assert.NotNil(storedPublicKeyBytes, "unexpectedly empty stored public key byte slice")
	assert.Equal(storedPublicKeyBytes, publicKeyBytes)

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestSystemSavePublicKeyInvalidKey() {
	assert := s.Assert()

	clientID := uuid.NewRandom().String()
	groupID := "T44444"

	// Setup Group and System
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	assert.Nil(err)
	system := System{ClientID: clientID, GroupID: groupID}
	err = s.db.Create(&system).Error
	assert.Nil(err)

	emptyPEM := "-----BEGIN RSA PUBLIC KEY-----    -----END RSA PUBLIC KEY-----"
	invalidPEM :=
		`-----BEGIN RSA PUBLIC KEY-----
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
OwIDAQAB
-----END RSA PUBLIC KEY-----`
	keyPair, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(err, "unable to generate key pair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	lowBitPubKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(lowBitPubKey, "unexpectedly empty public key byte slice")

	err = system.SavePublicKey(strings.NewReader(""))
	assert.NotNil(err, "empty string should not be saved")

	err = system.SavePublicKey(strings.NewReader(emptyPEM))
	assert.NotNil(err, "empty PEM should not be saved")

	err = system.SavePublicKey(strings.NewReader(invalidPEM))
	assert.NotNil(err, "invalid PEM should not be saved")

	err = system.SavePublicKey(bytes.NewReader(lowBitPubKey))
	assert.NotNil(err, "insecure public key should not be saved")

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestSystemPublicKeyEmpty() {
	assert := s.Assert()

	clientID := uuid.NewRandom().String()
	groupID := "T22222"

	// Setup Group and System
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	assert.Nil(err)
	system := System{ClientID: clientID, GroupID: groupID}
	err = s.db.Create(&system).Error
	assert.Nil(err)

	emptyPEM := "-----BEGIN RSA PUBLIC KEY-----    -----END RSA PUBLIC KEY-----"
	validPEM, err := generatePublicKey(2048)
	assert.Nil(err)

	err = system.SavePublicKey(strings.NewReader(""))
	assert.NotNil(err)
	k, err := system.GetPublicKey()
	assert.NotNil(err)
	assert.Nil(k, "Empty string does not yield nil public key!")
	err = system.SavePublicKey(strings.NewReader(emptyPEM))
	assert.NotNil(err)
	k, err = system.GetPublicKey()
	assert.NotNil(err)
	assert.Nil(k, "Empty PEM key does not yield nil public key!")
	err = system.SavePublicKey(strings.NewReader(validPEM))
	assert.Nil(err)
	k, err = system.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(k, "Valid PEM key yields nil public key!")

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

func (s *SystemsTestSuite) TestGetSystemByClientIDSuccess() {
	assert := s.Assert()

	group := Group{GroupID: "abcdef123456"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID, ClientID: "987654zyxwvu", ClientName: "Client with System"}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	sys, err := GetSystemByClientID(system.ClientID)
	assert.Nil(err)
	assert.NotEmpty(sys)
	assert.Equal("Client with System", sys.ClientName)

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestRegisterSystemSuccess() {
	assert := s.Assert()

	groupID := "T54321"
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey, err := generatePublicKey(2048)
	assert.Nil(err)

	creds, err := RegisterSystem("abcd1234", "Create System Test", "https://no.client.uri.net", groupID, DEFAULT_SCOPE, pubKey)
	assert.Nil(err)
	assert.Equal("Create System Test", creds.ClientName)
	assert.NotEqual("", creds.ClientSecret)

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestRegisterSystemMissingData() {
	assert := s.Assert()

	groupID := "T11223"
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey, err := generatePublicKey(2048)
	assert.Nil(err)

	// No clientID
	creds, err := RegisterSystem("", "Register System Failure", "https://no.client.uri.net", groupID, DEFAULT_SCOPE, pubKey)
	assert.NotNil(err)
	assert.Empty(creds)

	// No clientName
	creds, err = RegisterSystem("a1b2c3d3", "", "https://no.client.uri.net", groupID, DEFAULT_SCOPE, pubKey)
	assert.NotNil(err)
	assert.Empty(creds)

	// No clientURI = success
	creds, err = RegisterSystem("a1b2c3d4", "Register System Success1", "", groupID, DEFAULT_SCOPE, pubKey)
	assert.Nil(err)
	assert.NotEmpty(creds)

	// No scope = success
	creds, err = RegisterSystem("a1b2c3d5", "Register System Success2", "https://no.client.uri.net", groupID, "", pubKey)
	assert.Nil(err)
	assert.NotEmpty(creds)

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestRegisterSystemBadKey() {
	assert := s.Assert()

	groupID := "T22334"
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey, err := generatePublicKey(1024)
	assert.Nil(err)

	// Blank key
	creds, err := RegisterSystem("", "Register System Failure", "https://no.client.uri.net", groupID, DEFAULT_SCOPE, "")
	assert.NotNil(err)
	assert.Empty(creds)

	// Invalid key
	creds, err = RegisterSystem("", "Register System Failure", "https://no.client.uri.net", groupID, DEFAULT_SCOPE, "NotAKey")
	assert.NotNil(err)
	assert.Empty(creds)

	// Key length too low
	creds, err = RegisterSystem("", "Register System Failure", "https://no.client.uri.net", groupID, DEFAULT_SCOPE, pubKey)
	assert.NotNil(err)
	assert.Empty(creds)

	err = s.db.Unscoped().Delete(&group).Error
	assert.Nil(err)
}

func generatePublicKey(bits int) (string, error) {
	keyPair, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", fmt.Errorf("unable to generate keyPair: %s", err.Error())
	}

	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	if err != nil {
		return "", fmt.Errorf("unable to marshal public key: %s", err.Error())
	}

	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})

	return string(publicKeyBytes), nil
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
	secret1, err := GenerateSecret()

	if err != nil {
		s.FailNow("cannot generate random secret")
	}
	hashedSecret1, err := NewHash(secret1)
	if err != nil {
		s.FailNow("cannot hash random secret")
	}
	err = system.SaveSecret(hashedSecret1.String())
	if err != nil {
		s.FailNow(err.Error())
	}

	// Second secret should cause first secret to be soft-deleted
	secret2, err := GenerateSecret()
  
	if err != nil {
		s.FailNow("cannot generate random secret")
	}
	hashedSecret2, err := NewHash(secret2)
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
	assert.True(Hash(savedHash).IsHashOf(secret2))

	err = s.cleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) cleanDatabase(group Group) error {
	var system System
	var encryptionKey EncryptionKey
	var secret Secret
	var systemIds []int
  
	err := s.db.Table("systems").Where("group_id = ?", group.GroupID).Pluck("ID", &systemIds).Error
	if err != nil {
		return fmt.Errorf("unable to find associated systems: %s", err.Error())
	}
	fmt.Println("System ID's: ", systemIds)

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

func TestSystemsTestSuite(t *testing.T) {
	suite.Run(t, new(SystemsTestSuite))
}
