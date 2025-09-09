// Package encryption provides functions for encrypting and decrypting data using RSA and AES-GCM.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// HybridDecrypt расшифровывает гибридно зашифрованные данные
func HybridDecrypt(encryptedData []byte, rsaPrivateKeyPath string) ([]byte, error) {
	if rsaPrivateKeyPath == "" {
		return nil, errors.New("no private key path configured")
	}

	// Парсим JSON пакет
	var packet map[string]string
	if err := json.Unmarshal(encryptedData, &packet); err != nil {
		return nil, fmt.Errorf("failed to parse encrypted packet: %w", err)
	}

	// Извлекаем компоненты
	encryptedAESKeyBase64, ok := packet["key"]
	if !ok {
		return nil, errors.New("missing 'key' field in encrypted packet")
	}

	nonceBase64, ok := packet["nonce"]
	if !ok {
		return nil, errors.New("missing 'nonce' field in encrypted packet")
	}

	encryptedPayloadBase64, ok := packet["data"]
	if !ok {
		return nil, errors.New("missing 'data' field in encrypted packet")
	}

	// Декодируем из base64
	encryptedAESKey, err := base64.StdEncoding.DecodeString(encryptedAESKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	encryptedPayload, err := base64.StdEncoding.DecodeString(encryptedPayloadBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted payload: %w", err)
	}

	// Загружаем приватный RSA ключ
	privateKey, err := loadRSAPrivateKey(rsaPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load RSA private key: %w", err)
	}

	// Расшифровываем AES ключ с помощью RSA
	aesKey, err := rsa.DecryptOAEP(
		sha256Hash(),
		rand.Reader,
		privateKey,
		encryptedAESKey,
		nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decryption of AES key failed: %w", err)
	}

	// Расшифровываем данные с помощью AES-GCM
	decryptedPayload, err := decryptAESGCM(aesKey, nonce, encryptedPayload)
	if err != nil {
		return nil, fmt.Errorf("AES decryption failed: %w", err)
	}

	return decryptedPayload, nil
}

// loadRSAPrivateKey загружает RSA приватный ключ из файла PEM
func loadRSAPrivateKey(rsaPrivateKeyPath string) (*rsa.PrivateKey, error) {
	if rsaPrivateKeyPath == "" {
		return nil, errors.New("no private key path configured")
	}

	data, err := os.ReadFile(rsaPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Парсим PEM блок
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse PEM block from private key file")
	}

	// Пытаемся парсить как PKCS1
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	// Пытаемся парсить как PKCS8
	privInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key (PKCS1 and PKCS8): %w", err)
	}

	privKey, ok := privInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not RSA private key")
	}

	return privKey, nil
}

// decryptAESGCM расшифровывает данные data ключом aesKey и nonce с помощью AES-GCM
func decryptAESGCM(aesKey, nonce, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aesgcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
