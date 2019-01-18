package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
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
}

func main() {
	ek, err := hex.DecodeString(encryptedKey)
	if err != nil {
		panic(err)
	}
	pk := getPrivateKey(private)
	decryptedFile := decryptFile(pk, ek, filepath)
	fmt.Printf("Decrypted file: %s\n", decryptedFile)
}
