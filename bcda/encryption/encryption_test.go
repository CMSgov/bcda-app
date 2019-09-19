package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"io/ioutil"
	"os"
	"testing"
)

type EncryptionTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *EncryptionTestSuite) SetupTest() {
	models.InitializeGormModels()
	s.db = database.GetGORMDbConnection()
	os.Setenv("ATO_PUBLIC_KEY_FILE", "../../shared_files/ATO_public.pem")
	os.Setenv("ATO_PRIVATE_KEY_FILE", "../../shared_files/ATO_private.pem")
}

func (s *EncryptionTestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *EncryptionTestSuite) TestEncryptBytes() {
	// Make a random String for encrypting
	testBytes := []byte(uuid.NewRandom().String())
	// Encrypt the sting and get the key back
	encryptedBytes, encryptedKey, err := EncryptBytes(models.GetATOPublicKey(), testBytes, "TEST")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), encryptedBytes)
	assert.NotNil(s.T(), encryptedKey)
	// Make sure we changed something
	assert.NotEqual(s.T(), testBytes, encryptedBytes)
	// Decrypt the Key
	decryptedKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, models.GetATOPrivateKey(), encryptedKey, []byte("TEST"))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), decryptedKey)
	// Decrypted Key can not match the encrypted key
	assert.NotEqual(s.T(), encryptedKey, decryptedKey)
	// This is clunky, but apparently how fixed size arrays work :(
	key := [32]byte{}
	copy(key[:], decryptedKey[0:32])
	decryptedBytes, err := decrypt(encryptedBytes, &key)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), decryptedBytes)
	// Back to where we started
	assert.Equal(s.T(), testBytes, decryptedBytes)

}

func (s *EncryptionTestSuite) TestEncryptFile() {
	fromPath := "../../shared_files/synthetic_beneficiary_data"
	fileName := "Coverage"

	r, k, err := EncryptFile(fromPath, fileName, models.GetATOPublicKey())
	assert.Nil(s.T(), err)

	rawBytes, err := ioutil.ReadFile(fromPath + "/" + fileName)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), rawBytes, r)

	decryptedKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, models.GetATOPrivateKey(), k, []byte("Coverage"))
	assert.Nil(s.T(), err)
	key := [32]byte{}
	copy(key[:], decryptedKey[0:32])
	decryptedBytes, err := decrypt(r, &key)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), rawBytes, decryptedBytes)
}

func TestEncryptionTestSuite(t *testing.T) {
	suite.Run(t, new(EncryptionTestSuite))
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func decrypt(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}
