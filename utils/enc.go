package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

func Encrypt(text string, key []byte) (string, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", errors.New("encryption key must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := []byte(text)
	ciphertext := make([]byte, len(plaintext))

	stream := cipher.NewCFBEncrypter(block, key[:block.BlockSize()])
	stream.XORKeyStream(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encoded string, key []byte) (string, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", errors.New("decryption key must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	ciphertext, _ := base64.StdEncoding.DecodeString(encoded)

	plaintext := make([]byte, len(ciphertext))
	stream := cipher.NewCFBDecrypter(block, key[:block.BlockSize()])
	stream.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}
