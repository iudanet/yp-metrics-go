package server

import (
	"bytes"
	"io"
	"net/http"

	"github.com/iudanet/yp-metrics-go/internal/utils"
	"go.uber.org/zap"
)

func (s *Service) VerifyHash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.SginKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		// Read and restore the request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// Verify hash
		receivedHash := r.Header.Get("Hashsha256")
		if receivedHash == "" {
			http.Error(w, "Missing Hashsha256 header", http.StatusBadRequest)
			s.logger.Info("Missing Hashsha256 header")
			return
		}
		hash := utils.CalculateHash(bodyBytes, s.config.SginKey)
		s.logger.Info("colculete hash", zap.String("hash", hash))
		if receivedHash != hash {
			http.Error(w, "Invalid Hashsha256", http.StatusBadRequest)
			s.logger.Info("Invalid Hashsha256")
			return
		}

		next.ServeHTTP(w, r)
	})
}
