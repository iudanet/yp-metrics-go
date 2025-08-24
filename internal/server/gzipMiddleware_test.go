package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware(t *testing.T) {
	// Setup
	cfg := config.NewServerConfig()
	logger, _ := logger.New("Info")
	store := localStore.New()
	svc := NewService(store, cfg, logger, store)

	// Test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	middleware := svc.GzipMiddleware(handler)

	// Test data
	testBody := `{"id":"test","type":"gauge","value":10.5}`

	tests := []struct {
		name             string
		acceptEncoding   string
		contentEncoding  string
		requestBody      string
		expectedEncoding string
		compressRequest  bool
		expectCompressed bool
	}{
		{
			name:             "Client_supports_gzip",
			acceptEncoding:   "gzip",
			requestBody:      testBody,
			expectedEncoding: "gzip",
			expectCompressed: true,
		},
		{
			name:             "Client_supports_multiple_encodings",
			acceptEncoding:   "deflate, gzip, br",
			requestBody:      testBody,
			expectedEncoding: "gzip",
			expectCompressed: true,
		},
		{
			name:             "Client_does_not_support_compression",
			acceptEncoding:   "",
			requestBody:      testBody,
			expectedEncoding: "",
			expectCompressed: false,
		},
		{
			name:             "Compressed_request",
			contentEncoding:  "gzip",
			requestBody:      testBody,
			compressRequest:  true,
			acceptEncoding:   "gzip", // Добавляем поддержку сжатия для ответа
			expectedEncoding: "gzip",
			expectCompressed: true,
		},
		{
			name:             "Uncompressed_request",
			contentEncoding:  "",
			requestBody:      testBody,
			compressRequest:  false,
			expectedEncoding: "",
			expectCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var body io.Reader
			if tt.compressRequest {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				_, err := gz.Write([]byte(tt.requestBody))
				require.NoError(t, err)
				require.NoError(t, gz.Close())
				body = &buf
			} else {
				body = strings.NewReader(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/update/", body)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, http.StatusOK, rr.Code)

			// Check content encoding header
			contentEncoding := rr.Header().Get("Content-Encoding")
			assert.Equal(t, tt.expectedEncoding, contentEncoding)

			// Verify response body
			var responseBody []byte
			var err error

			if contentEncoding == "gzip" {
				var gz *gzip.Reader
				gz, err = gzip.NewReader(rr.Body)
				require.NoError(t, err)
				defer gz.Close()
				responseBody, err = io.ReadAll(gz)
				require.NoError(t, err)
			} else {
				responseBody, err = io.ReadAll(rr.Body)
				require.NoError(t, err)
			}

			assert.Equal(t, tt.requestBody, string(responseBody))
		})
	}
}

func TestGzipMiddleware_ContentTypes(t *testing.T) {
	// Setup
	cfg := config.NewServerConfig()
	logger, _ := logger.New("Info")
	store := localStore.New()
	svc := NewService(store, cfg, logger, store)

	// Test handler with different content types
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.URL.Query().Get("content-type")
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	})

	middleware := svc.GzipMiddleware(handler)

	tests := []struct {
		name           string
		contentType    string
		shouldCompress bool
	}{
		{"JSON", "application/json", true},
		{"HTML", "text/html", true},
		{"Plain text", "text/plain", false},
		{"Image", "image/png", false},
		// Убираем JavaScript, CSS, XML, так как они не должны сжиматься по ТЗ
		{"Octet stream", "application/octet-stream", false},
		{"With charset", "text/html; charset=utf-8", true},
		{"Invalid type", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Экранируем параметры URL
			u, err := url.Parse("/")
			require.NoError(t, err)
			q := u.Query()
			q.Set("content-type", tt.contentType)
			u.RawQuery = q.Encode()

			req := httptest.NewRequest("GET", u.String(), nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			contentEncoding := rr.Header().Get("Content-Encoding")
			body, err := io.ReadAll(rr.Body)
			require.NoError(t, err)

			if tt.shouldCompress {
				assert.Equal(t, "gzip", contentEncoding)
				// Проверяем, что тело сжато (не равно исходному)
				assert.NotEqual(t, "test content", string(body))

				// Декомпрессия для проверки содержимого
				if contentEncoding == "gzip" {
					gz, err := gzip.NewReader(bytes.NewReader(body))
					require.NoError(t, err)
					defer gz.Close()
					decompressed, err := io.ReadAll(gz)
					require.NoError(t, err)
					assert.Equal(t, "test content", string(decompressed))
				}
			} else {
				assert.Empty(t, contentEncoding)
				assert.Equal(t, "test content", string(body))
			}
		})
	}
}

func TestGzipMiddleware_ErrorHandling(t *testing.T) {
	// Setup
	cfg := config.NewServerConfig()
	logger, _ := logger.New("Info")
	store := localStore.New()
	svc := NewService(store, cfg, logger, store)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := svc.GzipMiddleware(handler)

	t.Run("Invalid gzip body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("invalid gzip data"))
		req.Header.Set("Content-Encoding", "gzip")

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
