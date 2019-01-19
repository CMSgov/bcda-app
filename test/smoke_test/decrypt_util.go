package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func decryptCipher(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
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

func decryptFile(privateKey *rsa.PrivateKey, encryptedKey []byte, filename string) string {
	base := path.Base(filename)
	decryptedKey, err := rsa.DecryptOAEP(
		sha256.New(), rand.Reader, privateKey, encryptedKey, []byte(base))
	if err != nil {
		panic(err)
	}

	ciphertext, err := ioutil.ReadFile(fmt.Sprintf(filename))
	if err != nil {
		panic(err)
	}

	var plaintext []byte
	key := [32]byte{}
	copy(key[:], decryptedKey[0:32])
	plaintext, err = decryptCipher(ciphertext, &key)
	if err != nil {
		panic(err)
	}

	decryptedFile := "/tmp/decrypted_" + base
	err = ioutil.WriteFile(decryptedFile, plaintext, 0644)
	if err != nil {
		panic(err)
	}

	return decryptedFile
}

func getPrivateKey(loc string) *rsa.PrivateKey {
	pkFile, err := os.Open(loc)
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := pkFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)
	buffer := bufio.NewReader(pkFile)

	_, err = buffer.Read(pembytes)
	if err != nil {
		log.Panic(err)
	}

	data, _ := pem.Decode([]byte(pembytes))
	pkFile.Close()

	imported, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	if err != nil {
		log.Panic(err)
	}

	return imported
}
