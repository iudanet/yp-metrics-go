package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
)

func TestCheckContentType(t *testing.T) {
	// Setup
	cfg := config.NewServerConfig()
	logger, _ := logger.New("Info")
	store := localStore.New()
	svc := NewService(store, cfg, logger, store)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := svc.CheckContentType(handler)

	tests := []struct {
		name           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Valid_application/json",
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid_with_charset",
			contentType:    "application/json; charset=utf-8",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid_with_version",
			contentType:    "application/json; version=1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid_text/plain",
			contentType:    "text/plain",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Invalid_multipart/form-data",
			contentType:    "multipart/form-data",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Missing_content_type",
			contentType:    "",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Invalid_prefix",
			contentType:    "application/xml",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Case_insensitive",
			contentType:    "APPLICATION/JSON",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/update/", strings.NewReader("{}"))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
