package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

var private, encryptedKey, filepath string

func init() {
	flag.StringVar(&encryptedKey, "key", "", "encrypted symmetric key used for file decryption (hex-encoded string)")
	flag.StringVar(&filepath, "file", "", "location of encrypted file")
	flag.StringVar(&private, "pk", "", "location of private key to use for decryption of symmetric key")
	flag.Parse()

	if encryptedKey == "" || filepath == "" || private == "" {
		fmt.Println("missing argument(s)")
		os.Exit(1)
	}
	r, _ := regexp.Compile("^[a-f0-9]{8}-?[a-f0-9]{4}-?4[a-f0-9]{3}-?[89ab][a-f0-9]{3}-?[a-f0-9]{12}")
	filename := path.Base(filepath)
	uuid := strings.Split(filename, ".")[0]
	if !r.MatchString(uuid) {
		fmt.Printf("File name does not appear to be valid.\nPlease use the exact file name from the job status endpoint (i.e., of the format: <UUID>.ndjson).\n")
		os.Exit(1)
	}
}

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

func decryptFile(privateKey *rsa.PrivateKey, encryptedKey []byte, filename string) {
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

	fmt.Printf("%s", plaintext)
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

func main() {
	ek, err := hex.DecodeString(encryptedKey)
	if err != nil {
		panic(err)
	}
	pk := getPrivateKey(private)
	decryptFile(pk, ek, filepath)
}
