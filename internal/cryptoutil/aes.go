// Package cryptoutil re-implements CryptoJS's AES.encrypt/decrypt(passphrase)
// so data encrypted by the original Node/CryptoJS backend stays decryptable,
// and vice versa. CryptoJS derives the key/IV with OpenSSL's EVP_BytesToKey
// (MD5) and prefixes the output with "Salted__" + 8-byte salt before base64.
package cryptoutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

const (
	saltLen = 8
	keyLen  = 32 // AES-256
	ivLen   = 16
)

func evpBytesToKey(passphrase, salt []byte) (key, iv []byte) {
	var (
		concatenated []byte
		prevDigest   []byte
	)
	for len(concatenated) < keyLen+ivLen {
		h := md5.New()
		h.Write(prevDigest)
		h.Write(passphrase)
		h.Write(salt)
		digest := h.Sum(nil)
		concatenated = append(concatenated, digest...)
		prevDigest = digest
	}
	return concatenated[:keyLen], concatenated[keyLen : keyLen+ivLen]
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("cryptoutil: empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > len(data) {
		return nil, errors.New("cryptoutil: invalid padding")
	}
	return data[:len(data)-padLen], nil
}

// Encrypt mirrors CryptoJS.AES.encrypt(plaintext, passphrase).toString().
func Encrypt(plaintext, passphrase string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key, iv := evpBytesToKey([]byte(passphrase), salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	out := append([]byte("Salted__"), salt...)
	out = append(out, ciphertext...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt mirrors CryptoJS.AES.decrypt(payload, passphrase).toString(CryptoJS.enc.Utf8).
func Decrypt(payload, passphrase string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	if len(raw) < saltLen+8 || string(raw[:8]) != "Salted__" {
		return "", errors.New("cryptoutil: missing Salted__ header")
	}

	salt := raw[8 : 8+saltLen]
	ciphertext := raw[8+saltLen:]
	if len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("cryptoutil: ciphertext is not a multiple of the block size")
	}

	key, iv := evpBytesToKey([]byte(passphrase), salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plainPadded := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plainPadded, ciphertext)

	plain, err := pkcs7Unpad(plainPadded)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
