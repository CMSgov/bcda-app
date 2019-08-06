package ssas

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

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
	system := System{GroupID: group.GroupID, ClientID: "test-revoke-system-key-pair-client"}
	s.db.Save(&system)
	encryptionKey := EncryptionKey{SystemID: system.ID}
	s.db.Save(&encryptionKey)

	err := system.RevokeSystemKeyPair()
	assert.Nil(err)
	assert.Empty(system.EncryptionKeys)
	s.db.Unscoped().Find(&encryptionKey)
	assert.NotNil(encryptionKey.DeletedAt)

	err = CleanDatabase(group)
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

	err = CleanDatabase(group)
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

	err = CleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestGetEncryptionKey() {
	group := Group{GroupID: "test-get-encryption-key-group"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey := `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
OwIDAQAB
-----END RSA PUBLIC KEY-----`

	origKey := EncryptionKey{
		SystemID: system.ID,
		Body:     pubKey,
	}
	err = s.db.Create(&origKey).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	key, err := system.GetEncryptionKey("")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), pubKey, key.Body)

	_ = CleanDatabase(group)
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
	storedKey, err := system.GetEncryptionKey("")
	assert.Nil(err)
	assert.NotNil(storedKey)
	assert.Equal(storedKey.Body, string(publicKeyBytes))

	err = CleanDatabase(group)
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

	err = CleanDatabase(group)
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
	assert.EqualError(err, fmt.Sprintf("invalid public key for clientID %s: not able to decode PEM-formatted public key", clientID))
	k, err := system.GetEncryptionKey("")
	assert.EqualError(err, fmt.Sprintf("cannot find key for clientID %s: record not found", clientID))
	assert.Empty(k, "Empty string does not yield empty encryption key!")
	err = system.SavePublicKey(strings.NewReader(emptyPEM))
	assert.EqualError(err, fmt.Sprintf("invalid public key for clientID %s: not able to decode PEM-formatted public key", clientID))
	k, err = system.GetEncryptionKey("")
	assert.EqualError(err, fmt.Sprintf("cannot find key for clientID %s: record not found", clientID))
	assert.Empty(k, "Empty PEM key does not yield empty encryption key!")
	err = system.SavePublicKey(strings.NewReader(validPEM))
	assert.Nil(err)
	k, err = system.GetEncryptionKey("")
	assert.Nil(err)
	assert.NotEmpty(k, "Valid PEM key yields empty public key!")

	err = CleanDatabase(group)
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

	err = CleanDatabase(group)
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

	err = CleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestSystemClientGroupDuplicate() {
	assert := s.Assert()

	group1 := Group{GroupID: "fabcde612345"}
	err := s.db.Create(&group1).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	group2 := Group{GroupID: "efabcd561234"}
	err = s.db.Create(&group2).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group1.GroupID, ClientID: "498765uzyxwv", ClientName: "First Client"}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system = System{GroupID: group2.GroupID, ClientID: "498765uzyxwv", ClientName: "Duplicate Client"}
	err = s.db.Create(&system).Error
	assert.EqualError(err, "pq: duplicate key value violates unique constraint \"idx_client\"")

	sys, err := GetSystemByClientID(system.ClientID)
	assert.Nil(err)
	assert.NotEmpty(sys)
	assert.Equal("First Client", sys.ClientName)

	err = CleanDatabase(group1)
	assert.Nil(err)

	err = CleanDatabase(group2)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestRegisterSystemSuccess() {
	assert := s.Assert()

	trackingID := uuid.NewRandom().String()
	groupID := "T54321"
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey, err := generatePublicKey(2048)
	assert.Nil(err)

	creds, err := RegisterSystem("Create System Test", groupID, DefaultScope, pubKey, trackingID)
	assert.Nil(err)
	assert.Equal("Create System Test", creds.ClientName)
	assert.NotEqual("", creds.ClientSecret)

	err = CleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestRegisterSystemMissingData() {
	assert := s.Assert()

	trackingID := uuid.NewRandom().String()
	groupID := "T11223"
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey, err := generatePublicKey(2048)
	assert.Nil(err)

	// No clientName
	creds, err := RegisterSystem("", groupID, DefaultScope, pubKey, trackingID)
	assert.EqualError(err, "clientName is required")
	assert.Empty(creds)

	// No scope = success
	creds, err = RegisterSystem("Register System Success2", groupID, "", pubKey, trackingID)
	assert.Nil(err)
	assert.NotEmpty(creds)

	err = CleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestRegisterSystemBadKey() {
	assert := s.Assert()

	trackingID := uuid.NewRandom().String()
	groupID := "T22334"
	group := Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	pubKey, err := generatePublicKey(1024)
	assert.Nil(err)

	// Blank key
	creds, err := RegisterSystem("Register System Failure", groupID, DefaultScope, "", trackingID)
	assert.EqualError(err, "error in public key")
	assert.Empty(creds)

	// Invalid key
	creds, err = RegisterSystem("Register System Failure", groupID, DefaultScope, "NotAKey", trackingID)
	assert.EqualError(err, "error in public key")
	assert.Empty(creds)

	// Key length too low
	creds, err = RegisterSystem("Register System Failure", groupID, DefaultScope, pubKey, trackingID)
	assert.EqualError(err, "error in public key")
	assert.Empty(creds)

	err = s.db.Unscoped().Delete(&group).Error
	assert.Nil(err)
}

func generatePublicKey(bits int) (string, error) {
	return GeneratePublicKey(bits)
}

func (s *SystemsTestSuite) TestSaveSecret() {
	assert := s.Assert()

	group := Group{GroupID: "T21212"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID, ClientID: "test-save-secret-client"}
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

	err = CleanDatabase(group)
	assert.Nil(err)
}

func (s *SystemsTestSuite) TestDeactivateSecrets() {
	group := Group{GroupID: "test-deactivate-secrets-group"}
	s.db.Create(&group)
	system := System{GroupID: group.GroupID, ClientID: "test-deactivate-secrets-client"}
	s.db.Create(&system)
	secret := Secret{Hash: "test-deactivate-secrets-hash", SystemID: system.ID}
	s.db.Create(&secret)

	var systemSecrets []Secret
	s.db.Find(&systemSecrets, "system_id = ?", system.ID)
	assert.NotEmpty(s.T(), systemSecrets)

	err := system.DeactivateSecrets()
	assert.Nil(s.T(), err)
	s.db.Find(&systemSecrets, "system_id = ?", system.ID)
	assert.Empty(s.T(), systemSecrets)

	_ = CleanDatabase(group)
}

func (s *SystemsTestSuite) TestResetSecret() {
	group := Group{GroupID: "group-12345"}
	s.db.Create(&group)
	system := System{GroupID: group.GroupID, ClientID: "client-12345"}
	s.db.Create(&system)
	secret := Secret{Hash: "foo", SystemID: system.ID}
	s.db.Create(&secret)

	secret1 := Secret{}
	s.db.Where("system_id = ?", system.ID).First(&secret1)
	assert.Equal(s.T(), secret1.Hash, secret.Hash)

	credentials, err := system.ResetSecret("tracking-id")
	if err != nil {
		s.FailNow("Error from ResetSecret()", err.Error())
		return
	}

	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), credentials)
	assert.NotEqual(s.T(), secret1.Hash, credentials.ClientSecret)

	_ = CleanDatabase(group)
}

func (s *SystemsTestSuite) TestScopeEnvSuccess() {
	key := "SSAS_DEFAULT_SYSTEM_SCOPE"
	newScope := "my_scope"
	oldScope := os.Getenv(key)
	err := os.Setenv(key, newScope)
	if err != nil {
		s.FailNow(err.Error())
	}
	getEnvVars()

	assert.Equal(s.T(), newScope, DefaultScope)
	err = os.Setenv(key, oldScope)
	assert.Nil(s.T(), err)
}

func (s *SystemsTestSuite) TestScopeEnvFailure() {
	scope := ""
	err := os.Setenv("SSAS_DEFAULT_SYSTEM_SCOPE", scope)
	if err != nil {
		s.FailNow(err.Error())
	}

	assert.Panics(s.T(), func() { getEnvVars() })
}

func TestSystemsTestSuite(t *testing.T) {
	suite.Run(t, new(SystemsTestSuite))
}
