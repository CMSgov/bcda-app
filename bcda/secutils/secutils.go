package secutils

import (
	"bufio"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
)

func OpenPrivateKeyFile(privateKeyFile *os.File) *rsa.PrivateKey {
	pemfileinfo, err := privateKeyFile.Stat()
	if err != nil {
		log.Panic(err)
	}
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)
	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)
	if err != nil {
		// Above buffer.Read succeeded on a blank file Not Sure how to reach this
		log.Panic(err)
	}

	data, _ := pem.Decode([]byte(pembytes))
	err = privateKeyFile.Close()
	if err != nil {
		log.Panic(err)
	}

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	if err != nil {
		// Above function panicked when receiving a bad and blank key file.  This may be unreachable
		log.Panic(err)
	}

	return privateKeyImported
}

func OpenPublicKeyFile(publicKeyFile *os.File) *rsa.PublicKey {
	pemfileinfo, err := publicKeyFile.Stat()
	if err != nil {
		log.Panic(err)
	}
	var size int64 = pemfileinfo.Size()
	pemBytes := make([]byte, size)
	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pemBytes)
	if err != nil {
		log.Fatal(err)
	}

	data, _ := pem.Decode([]byte(pemBytes))

	err = publicKeyFile.Close()
	if err != nil {
		log.Panic(err)
	}

	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)

	if !ok {
		panic(err)
	}
	return rsaPub
}

