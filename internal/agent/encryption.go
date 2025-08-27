package agent

import (
	"bytes"
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
	"net/http"
	"os"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/retry"
	"github.com/iudanet/yp-metrics-go/internal/utils"
)

// loadRSAPublicKey загружает RSA публичный ключ из файла PEM (PKCS1/PKIX)
func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pubInterface, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		// Может быть PEM с PKCS1
		pubKey, err2 := x509.ParsePKCS1PublicKey(data)
		if err2 == nil {
			return pubKey, nil
		}
		return nil, err
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

// encryptHybrid шифрует входной payload гибридно и возвращает байты, которые можно отправить
func (a *Agent) encryptHybrid(jsonData []byte) ([]byte, error) {
	if a.config.RSAPublicKeyPath == "" {
		return nil, errors.New("no public key path configured")
	}
	pubKey, err := loadRSAPublicKey(a.config.RSAPublicKeyPath)
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

	// Форматируем итоговый пакет, например JSON:
	packet := map[string]string{
		"key":   base64.StdEncoding.EncodeToString(encryptedAESKey),
		"nonce": base64.StdEncoding.EncodeToString(nonce),
		"data":  base64.StdEncoding.EncodeToString(encryptedPayload),
	}
	return json.Marshal(packet)
}

func sha256Hash() hash.Hash {
	return crypto.SHA256.New()
}

// sendEncryptedRequest отправляет гибридно зашифрованные метрики
func (a *Agent) sendEncryptedMetric(metric *models.Metrics) error {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	encryptedData, err := a.encryptHybrid(jsonData)
	if err != nil {
		return err
	}

	// Сжимаем encryptedData gzip
	compressedData, err := compressData(encryptedData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/update/encrypted", a.config.MetricServerHost)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if a.config.SginKey != "" {
		hash := utils.CalculateHash(encryptedData, a.config.SginKey)
		req.Header.Set("HashSHA256", hash)
	}

	return retry.WithRetry(func() error {
		return a.sendRequest(req)
	})
}
