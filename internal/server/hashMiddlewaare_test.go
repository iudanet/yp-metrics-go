package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/iudanet/yp-metrics-go/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestVerifyHashMiddleware(t *testing.T) {
	// Setup with key
	cfg := &config.ServerConfig{SginKey: "test-key"}
	logger, _ := logger.New("Info")
	store := localStore.New()
	svc := NewService(store, cfg, logger, store)

	// Test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := svc.VerifyHash(handler)

	// Test data
	testBody := `{"id":"test","type":"gauge","value":10.5}`
	validHash := utils.CalculateHash([]byte(testBody), "test-key")

	tests := []struct {
		name           string
		hashHeader     string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Valid hash",
			hashHeader:     validHash,
			requestBody:    testBody,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid hash",
			hashHeader:     "invalid-hash",
			requestBody:    testBody,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing header",
			hashHeader:     "",
			requestBody:    testBody,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Different body",
			hashHeader:     validHash,
			requestBody:    `{"id":"different","type":"counter","delta":5}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty body",
			hashHeader:     utils.CalculateHash([]byte(""), "test-key"),
			requestBody:    "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/update/", bytes.NewReader([]byte(tt.requestBody)))
			if tt.hashHeader != "" {
				req.Header.Set("HashSHA256", tt.hashHeader)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}

	t.Run("No key configured", func(t *testing.T) {
		// Setup without key
		cfgNoKey := &config.ServerConfig{SginKey: ""}
		svcNoKey := NewService(store, cfgNoKey, logger, store)
		middlewareNoKey := svcNoKey.VerifyHash(handler)

		req := httptest.NewRequest("POST", "/update/", bytes.NewReader([]byte(testBody)))
		rr := httptest.NewRecorder()

		middlewareNoKey.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
