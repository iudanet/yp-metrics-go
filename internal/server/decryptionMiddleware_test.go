package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/encryption"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptionMiddleware(t *testing.T) {
	// Создаем временную RSA пару ключей для тестов
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Сохраняем приватный ключ во временный файл
	tmpPrivateFile, err := os.CreateTemp("", "test_private_key_*.pem")
	require.NoError(t, err)
	defer os.Remove(tmpPrivateFile.Name())

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	err = pem.Encode(tmpPrivateFile, privateKeyBlock)
	require.NoError(t, err)
	tmpPrivateFile.Close()

	// Сохраняем публичный ключ во временный файл для шифрования
	tmpPublicFile, err := os.CreateTemp("", "test_public_key_*.pem")
	require.NoError(t, err)
	defer os.Remove(tmpPublicFile.Name())

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	err = pem.Encode(tmpPublicFile, publicKeyBlock)
	require.NoError(t, err)
	tmpPublicFile.Close()

	// Тестовые данные
	testData := []byte(`{"id":"test","type":"gauge","value":42.5}`)
	encryptedData, err := encryption.Hybrid(testData, tmpPublicFile.Name())
	require.NoError(t, err)

	// Setup
	logger, _ := logger.New("Info")
	store := localStore.New()
	cfg := &config.ServerConfig{
		RSAPrivateKeyPath: tmpPrivateFile.Name(),
	}
	svc := NewService(store, cfg, logger, store)

	// Test handler для проверки расшифрованных данных
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	middleware := svc.DecryptionMiddleware(handler)

	tests := []struct {
		name           string
		contentType    string
		requestBody    []byte
		expectedBody   []byte
		expectedStatus int
	}{
		{
			name:           "encrypted_data",
			requestBody:    encryptedData,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedBody:   testData,
		},
		{
			name:           "plain_json_data",
			requestBody:    testData,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedBody:   testData,
		},
		{
			name:           "non_json_content_type",
			requestBody:    encryptedData,
			contentType:    "text/plain",
			expectedStatus: http.StatusOK,
			expectedBody:   encryptedData,
		},
		{
			name:           "invalid_encrypted_data",
			requestBody:    []byte(`{"key":"invalid","nonce":"invalid","data":"invalid"}`),
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/update/", bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, string(tt.expectedBody), rr.Body.String())
			}
		})
	}

	t.Run("no_private_key_configured", func(t *testing.T) {
		cfgNoKey := &config.ServerConfig{
			RSAPrivateKeyPath: "",
		}
		svcNoKey := NewService(store, cfgNoKey, logger, store)
		middlewareNoKey := svcNoKey.DecryptionMiddleware(handler)

		req := httptest.NewRequest("POST", "/update/", bytes.NewReader(encryptedData))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		middlewareNoKey.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		// Данные должны остаться зашифрованными
		assert.Equal(t, string(encryptedData), rr.Body.String())
	})
}
