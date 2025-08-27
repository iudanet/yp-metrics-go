package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/iudanet/yp-metrics-go/internal/encryption"
	"go.uber.org/zap"
)

// DecryptionMiddleware расшифровывает входящие запросы, если настроен приватный ключ
func (s *Service) DecryptionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если приватный ключ не настроен, пропускаем middleware
		if s.config.RSAPrivateKeyPath == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Проверяем Content-Type для зашифрованных данных
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			next.ServeHTTP(w, r)
			return
		}

		// Читаем тело запроса
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.logger.Error("Failed to read request body", zap.Error(err))
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Пытаемся определить, является ли тело зашифрованным пакетом
		var encryptedPacket map[string]string
		if err = json.Unmarshal(bodyBytes, &encryptedPacket); err != nil {
			// Если не JSON или не соответствует формату зашифрованного пакета, пропускаем
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		// Проверяем наличие полей зашифрованного пакета
		if _, hasKey := encryptedPacket["key"]; !hasKey {
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}
		if _, hasNonce := encryptedPacket["nonce"]; !hasNonce {
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}
		if _, hasData := encryptedPacket["data"]; !hasData {
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		// Расшифровываем данные
		decryptedData, err := encryption.HybridDecrypt(bodyBytes, s.config.RSAPrivateKeyPath)
		if err != nil {
			s.logger.Error("Failed to decrypt request", zap.Error(err))
			http.Error(w, "Failed to decrypt request", http.StatusBadRequest)
			return
		}

		s.logger.Info("Successfully decrypted request")

		// Заменяем тело запроса на расшифрованные данные
		r.Body = io.NopCloser(bytes.NewBuffer(decryptedData))
		r.ContentLength = int64(len(decryptedData))

		next.ServeHTTP(w, r)
	})
}
