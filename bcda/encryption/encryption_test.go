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
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"io/ioutil"
	"log"
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

func (s *EncryptionTestSuite) TestEncryptAndMove() {
	fromPath := "../../shared_files/synthetic_beneficiary_data"
	toPath := "../../shared_files/synthetic_beneficiary_data/encrypted_files"
	// This dir might not exist, need to make it
	if _, err := os.Stat(toPath); os.IsNotExist(err) {
		err = os.MkdirAll(toPath, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	fileName := "Coverage"
	j := models.Job{
		AcoID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Pending",
	}
	s.db.Save(&j)
	// Do the Encrypt and Move
	err := EncryptAndMove(fromPath, toPath, fileName, models.GetATOPublicKey(), j.ID)
	// No Errors
	assert.Nil(s.T(), err)
	// Should have some Job Keys
	// Get the key back from the Job
	var jobKey models.JobKey
	if s.db.First(&jobKey, "job_id = ?", j.ID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("unable to find JobKey"))
	}

	// Check that we have data for each job key
	for _, jobKey := range j.JobKeys {
		testUtils.PrintSeparator()
		assert.NotNil(s.T(), jobKey.EncryptedKey)
		assert.Equal(s.T(), "Coverage", jobKey.FileName)
		testUtils.PrintSeparator()
	}
	// Open up the encrypted file
	encryptedBytes, err := ioutil.ReadFile(toPath + "/" + fileName)
	assert.Nil(s.T(), err)
	// Open up the Raw file
	rawBytes, err := ioutil.ReadFile(fromPath + "/" + fileName)
	assert.Nil(s.T(), err)
	// Encrypted and Raw can't match
	assert.NotEqual(s.T(), rawBytes, encryptedBytes)
	// Get the key back from the Job

	decryptedKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, models.GetATOPrivateKey(), jobKey.EncryptedKey, []byte("Coverage"))
	assert.Nil(s.T(), err)
	key := [32]byte{}
	copy(key[:], decryptedKey[0:32])
	// Decrypt the file
	decryptedBytes, err := decrypt(encryptedBytes, &key)
	assert.Nil(s.T(), err)
	// Should be the same as before
	assert.Equal(s.T(), rawBytes, decryptedBytes)

	os.Remove(toPath + "/" + fileName)

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
