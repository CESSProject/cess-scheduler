package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

const (
	ivaes = "abcdefghijklmnopqrstuvwxyz123456"
	ivdes = "wumansgy"
)

var (
	ErrKeyLengthSixteen = errors.New("a sixteen or twenty-four or thirty-two length secret key is required")
	ErrIvAes            = errors.New("a sixteen-length ivaes is required")
)

func AesCtrEncrypt(plainText, key, ivAes []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrKeyLengthSixteen
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	var iv []byte
	if len(ivAes) != 0 {
		if len(ivAes) != 16 {
			return nil, ErrIvAes
		} else {
			iv = ivAes
		}
	} else {
		iv = []byte(ivaes)
	}
	stream := cipher.NewCTR(block, iv)

	cipherText := make([]byte, len(plainText))
	stream.XORKeyStream(cipherText, plainText)

	return cipherText, nil
}

func AesCtrDecrypt(cipherText, key, ivAes []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrKeyLengthSixteen
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	var iv []byte
	if len(ivAes) != 0 {
		if len(ivAes) != 16 {
			return nil, ErrIvAes
		} else {
			iv = ivAes
		}
	} else {
		iv = []byte(ivaes)
	}
	stream := cipher.NewCTR(block, iv)

	plainText := make([]byte, len(cipherText))
	stream.XORKeyStream(plainText, cipherText)

	return plainText, nil
}
