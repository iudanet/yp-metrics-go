package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"go.uber.org/zap"
)

func TestVerifyIPMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name           string
		trustedSubnet  string
		headerIP       string
		remoteAddr     string
		expectedStatus int
		shouldLogError bool
	}{
		{
			name:           "trusted IP should pass",
			trustedSubnet:  "192.168.1.0/24",
			headerIP:       "192.168.1.10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "untrusted IP should be forbidden",
			trustedSubnet:  "192.168.1.0/24",
			headerIP:       "10.0.0.1",
			expectedStatus: http.StatusForbidden,
			shouldLogError: true,
		},
		{
			name:           "invalid IP should be forbidden",
			trustedSubnet:  "192.168.1.0/24",
			headerIP:       "invalid-ip",
			expectedStatus: http.StatusForbidden,
			shouldLogError: true,
		},
		{
			name:           "no trusted subnet should allow all",
			trustedSubnet:  "0.0.0.0/0",
			headerIP:       "10.0.0.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "fallback to RemoteAddr when header missing",
			trustedSubnet:  "192.168.1.0/24",
			headerIP:       "",
			remoteAddr:     "192.168.1.20:8080",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid RemoteAddr should be forbidden",
			trustedSubnet:  "192.168.1.0/24",
			headerIP:       "",
			remoteAddr:     "invalid-addr",
			expectedStatus: http.StatusForbidden,
			shouldLogError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем mock сервис

			ipnets := config.IPNets{}
			ipnets.Set(tt.trustedSubnet)
			service := &Service{
				logger: logger,
				config: &config.ServerConfig{
					TrustedSubnet: ipnets,
				},
			}

			// Создаем тестовый handler
			handler := service.VerifyIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Создаем запрос
			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerIP != "" {
				req.Header.Set("X-Real-IP", tt.headerIP)
			}
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			// Выполняем запрос
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Проверяем статус
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestVerifyIPMiddleware_EdgeCases(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("IPv6 support", func(t *testing.T) {
		netips := config.IPNets{}
		netips.Set("2001:db8::/32")
		service := &Service{
			logger: logger,
			config: &config.ServerConfig{
				TrustedSubnet: netips,
			},
		}

		handler := service.VerifyIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "2001:db8::1")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("IPv6 should be supported, got status %d", rr.Code)
		}
	})

	t.Run("empty config", func(t *testing.T) {
		service := &Service{
			logger: logger,
			config: &config.ServerConfig{}, // TrustedSubnet is nil
		}

		handler := service.VerifyIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Should allow all when no trusted subnet configured, got status %d", rr.Code)
		}
	})
}
