package encryption

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// Parse private key file
func GetRSAPrivateKey(path string) *rsa.PrivateKey {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	info, _ := file.Stat()
	buf := make([]byte, info.Size())
	file.Read(buf)
	block, _ := pem.Decode(buf)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	return privateKey
}

// Parse public key file
func GetRSAPublicKey(path string) *rsa.PublicKey {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	info, _ := file.Stat()
	buf := make([]byte, info.Size())
	file.Read(buf)
	block, _ := pem.Decode(buf)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	publicKey := publicKeyInterface.(*rsa.PublicKey)
	return publicKey
}

// Parse private key
func ParsePrivateKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	return privateKey, err
}

// Parse public key
func ParsePublicKey(key []byte) (*rsa.PublicKey, error) {
	if len(key) == 0 {
		return nil, errors.New("Invalid key")
	}
	block, _ := pem.Decode(key)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey := publicKeyInterface.(*rsa.PublicKey)
	return publicKey, nil
}

// Calculate the signature
func CalcSign(msg []byte, privkey *rsa.PrivateKey) ([]byte, error) {
	hash := sha256.New()
	hash.Write(msg)
	bytes := hash.Sum(nil)
	sign, err := rsa.SignPKCS1v15(rand.Reader, privkey, crypto.SHA256, bytes)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

// Verify signature
func VerifySign(msg []byte, sign []byte, pubkey *rsa.PublicKey) bool {
	hash := sha256.New()
	hash.Write(msg)
	bytes := hash.Sum(nil)
	err := rsa.VerifyPKCS1v15(pubkey, crypto.SHA256, bytes, sign)
	return err == nil
}
