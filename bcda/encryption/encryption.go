package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

// Code in this file borrows heavily from https://github.com/gtank/cryptopasta

// NewEncryptionKey generates a random 256-bit key for Encrypt() and
// Decrypt(). It panics if the source of randomness fails.
func newEncryptionKey() *[32]byte {
	key := [32]byte{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}
	return &key
}

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func encrypt(plaintext []byte, key *[32]byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	ret := gcm.Seal(nonce, nonce, plaintext, nil)
	return ret, nil
}

func EncryptBytes(publicKey *rsa.PublicKey, plaintext []byte, label string) (ciphertext []byte, encryptedKey []byte, err error) {
	// 64GB is the max file size we can do.  64*1024^3 = 64 GB
	if len(plaintext) > 64*1024*1024*1024 {
		return nil, nil, errors.New("Max File size of 64 GB exceeded.  Unable to encrypt file.")
	}
	symmetricKey := newEncryptionKey()
	ciphertext, err = encrypt(plaintext, symmetricKey)
	if err != nil {
		return nil, nil, err
	}
	encryptedKey, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, symmetricKey[:], []byte(label))

	if err != nil {
		return nil, nil, err
	} else {
		return ciphertext, encryptedKey, nil
	}
}

func EncryptFile(path, name string, key *rsa.PublicKey) ([]byte, []byte, error) {
	/*#nosec*/
	fileBytes, err := ioutil.ReadFile(path + "/" + name)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	return EncryptBytes(key, fileBytes, name)
}
