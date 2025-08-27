// Package encryption provides functions for encrypting and decrypting data using hybrid encryption.
package encryption

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
)

// Hybrid шифрует входной payload гибридно и возвращает байты, которые можно отправить
func Hybrid(jsonData []byte, rsaPublicKeyPath string) ([]byte, error) {
	pubKey, err := loadRSAPublicKey(rsaPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load RSA public key: %w", err)
	}

	// Генерируем случайный AES ключ 32 байта
	aesKey := make([]byte, 32)
	if _, err = rand.Read(aesKey); err != nil {
		return nil, err
	}

	// Шифруем метрики AES-GCM
	nonce, encryptedPayload, err := encryptAESGCM(aesKey, jsonData)
	if err != nil {
		return nil, fmt.Errorf("AES encryption failed: %w", err)
	}

	// Шифруем AES ключ RSA публичным ключом
	encryptedAESKey, err := rsa.EncryptOAEP(
		sha256Hash(), // хеш для OAEP
		rand.Reader,
		pubKey,
		aesKey,
		nil)
	if err != nil {
		return nil, fmt.Errorf("RSA encryption of AES key failed: %w", err)
	}

	// Форматируем итоговый пакет в JSON
	packet := map[string]string{
		"key":   base64.StdEncoding.EncodeToString(encryptedAESKey),
		"nonce": base64.StdEncoding.EncodeToString(nonce),
		"data":  base64.StdEncoding.EncodeToString(encryptedPayload),
	}
	return json.Marshal(packet)
}

// loadRSAPublicKey загружает RSA публичный ключ из файла PEM (PKCS1/PKIX)
func loadRSAPublicKey(rsaPublicKeyPath string) (*rsa.PublicKey, error) {
	if rsaPublicKeyPath == "" {
		return nil, errors.New("no public key path configured")
	}

	data, err := os.ReadFile(rsaPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	pubInterface, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		// Может быть PEM с PKCS1
		pubKey, err2 := x509.ParsePKCS1PublicKey(data)
		if err2 == nil {
			return pubKey, nil
		}
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pubKey, nil
}

// encryptAESGCM шифрует данные data ключом aesKey с помощью AES-GCM
func encryptAESGCM(aesKey, data []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	return nonce, ciphertext, nil
}

func sha256Hash() hash.Hash {
	return crypto.SHA256.New()
}
