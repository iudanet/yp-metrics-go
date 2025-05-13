package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compressString compresses a string using gzip
func compressString(t *testing.T, data string) []byte {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	_, err := gzWriter.Write([]byte(data))
	require.NoError(t, err)

	err = gzWriter.Close()
	require.NoError(t, err)

	return buf.Bytes()
}

// decompressBody decompresses a response body
func decompressBody(t *testing.T, body io.ReadCloser) string {
	gzReader, err := gzip.NewReader(body)
	require.NoError(t, err)
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	return string(decompressed)
}

// setupTestService creates a test service with the gzip middleware
func setupTestService(t *testing.T) *service {
	newLogger, err := logger.New("Info")
	require.NoError(t, err)

	store := storage.NewStorage()
	cfg := config.NewServerConfig()

	return NewService(store, cfg, newLogger)
}

func TestGzipCompression(t *testing.T) {
	svc := setupTestService(t)

	// Create a test handler that returns a specific content type and body
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","message":"test response"}`))
	})

	// Wrap the handler with our gzip middleware
	handler := svc.GzipMiddleware(testHandler)

	tests := []struct {
		name           string
		acceptEncoding string
		wantCompressed bool
	}{
		{
			name:           "client_supports_gzip",
			acceptEncoding: "gzip",
			wantCompressed: true,
		},
		{
			name:           "client_does_not_support_gzip",
			acceptEncoding: "",
			wantCompressed: false,
		},
		{
			name:           "client_supports_multiple_encodings",
			acceptEncoding: "deflate, gzip, br",
			wantCompressed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the specified Accept-Encoding header
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			// Record the response
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Verify response status
			assert.Equal(t, http.StatusOK, recorder.Code)

			// Check if response was compressed based on Content-Encoding header
			contentEncoding := recorder.Header().Get("Content-Encoding")
			wasCompressed := contentEncoding == "gzip"
			assert.Equal(t, tt.wantCompressed, wasCompressed)

			// Verify the content was handled correctly
			if wasCompressed {
				// Decompress response body and verify content
				decompressed := decompressBody(t, recorder.Result().Body)
				assert.Equal(t, `{"status":"ok","message":"test response"}`, decompressed)
			} else {
				// Verify uncompressed content directly
				assert.Equal(t, `{"status":"ok","message":"test response"}`, recorder.Body.String())
			}
		})
	}
}

func TestGzipDecompression(t *testing.T) {
	svc := setupTestService(t)

	// Create a handler that reads and echoes the request body
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	// Wrap the handler with our gzip middleware
	handler := svc.GzipMiddleware(testHandler)

	testContent := "This is test content for compression"
	compressedContent := compressString(t, testContent)

	tests := []struct {
		name            string
		body            []byte
		contentEncoding string
		wantContent     string
	}{
		{
			name:            "compressed_request",
			body:            compressedContent,
			contentEncoding: "gzip",
			wantContent:     testContent,
		},
		{
			name:            "uncompressed_request",
			body:            []byte(testContent),
			contentEncoding: "",
			wantContent:     testContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the test body
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tt.body))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			// Record the response
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Verify response status
			assert.Equal(t, http.StatusOK, recorder.Code)

			// Verify that the middleware correctly decompressed the body
			assert.Equal(t, tt.wantContent, recorder.Body.String())
		})
	}
}

func TestGzipContentTypeFiltering(t *testing.T) {
	svc := setupTestService(t)

	tests := []struct {
		name           string
		contentType    string
		shouldCompress bool
	}{
		{
			name:           "application_json",
			contentType:    "application/json",
			shouldCompress: true,
		},
		{
			name:           "text_html",
			contentType:    "text/html",
			shouldCompress: true,
		},
		{
			name:           "text_plain",
			contentType:    "text/plain",
			shouldCompress: false,
		},
		{
			name:           "application_octet_stream",
			contentType:    "application/octet-stream",
			shouldCompress: false,
		},
		{
			name:           "image_png",
			contentType:    "image/png",
			shouldCompress: false,
		},
		{
			name:           "text_html_with_charset",
			contentType:    "text/html; charset=utf-8",
			shouldCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a handler that sets a specific content type
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test content"))
			})

			// Wrap with gzip middleware
			handler := svc.GzipMiddleware(testHandler)

			// Create request with gzip accept-encoding
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			// Record response
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Check if response was compressed
			contentEncoding := recorder.Header().Get("Content-Encoding")
			wasCompressed := contentEncoding == "gzip"

			assert.Equal(t, tt.shouldCompress, wasCompressed,
				"Content type %s should%s be compressed",
				tt.contentType,
				map[bool]string{true: "", false: " not"}[tt.shouldCompress])
		})
	}
}

func TestGzipErrorHandling(t *testing.T) {
	svc := setupTestService(t)

	tests := []struct {
		name           string
		statusCode     int
		shouldCompress bool
	}{
		{
			name:           "success_response",
			statusCode:     200,
			shouldCompress: true,
		},
		{
			name:           "redirect_response",
			statusCode:     301,
			shouldCompress: false,
		},
		{
			name:           "client_error",
			statusCode:     400,
			shouldCompress: false,
		},
		{
			name:           "server_error",
			statusCode:     500,
			shouldCompress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a specific test handler that returns the configured status code
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set Content-Type before WriteHeader
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"message":"test response"}`))
			})

			// Create a new test server with our handler
			server := httptest.NewServer(svc.GzipMiddleware(testHandler))
			defer server.Close()

			// Create a request with Accept-Encoding: gzip
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			req.Header.Set("Accept-Encoding", "gzip")

			// Send the request
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // Don't follow redirects
				},
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the status code
			assert.Equal(t, tt.statusCode, resp.StatusCode)

			// Check if response was compressed
			contentEncoding := resp.Header.Get("Content-Encoding")
			wasCompressed := contentEncoding == "gzip"

			assert.Equal(t, tt.shouldCompress, wasCompressed,
				"Status %d should%s be compressed",
				tt.statusCode,
				map[bool]string{true: "", false: " not"}[tt.shouldCompress])
		})
	}

	// Also test the direct compressWriter behavior
	t.Run("compressWriter_status_handling", func(t *testing.T) {
		for _, statusCode := range []int{200, 301, 400, 500} {
			t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
				recorder := httptest.NewRecorder()
				compWriter := newCompressWriter(recorder)

				// Set content type and status code
				compWriter.Header().Set("Content-Type", "application/json")
				compWriter.WriteHeader(statusCode)

				// Write content
				compWriter.Write([]byte(`{"test":"data"}`))

				// Close the writer
				err := compWriter.Close()
				require.NoError(t, err)

				// Check if compression was applied based on status code
				contentEncoding := recorder.Header().Get("Content-Encoding")
				wasCompressed := contentEncoding == "gzip"
				shouldCompress := statusCode < 300

				// Assert the expected behavior
				assert.Equal(t, shouldCompress, wasCompressed, "Status %d compression behavior", statusCode)
			})
		}
	})
}

func TestIsCompressibleType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "text_plain",
			contentType: "text/plain",
			want:        false,
		},
		{
			name:        "application_json",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "text_html_with_charset",
			contentType: "text/html; charset=utf-8",
			want:        true,
		},
		{
			name:        "image_png",
			contentType: "image/png",
			want:        false,
		},
		{
			name:        "empty_content_type",
			contentType: "",
			want:        false,
		},
		{
			name:        "audio_mpeg",
			contentType: "audio/mpeg",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCompressibleType(tt.contentType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompressWriterImplementsResponseWriter(t *testing.T) {
	// This test verifies that compressWriter properly implements http.ResponseWriter
	writer := httptest.NewRecorder()
	compWriter := newCompressWriter(writer)

	// Test that we can set headers
	compWriter.Header().Set("X-Test", "test-value")
	assert.Equal(t, "test-value", writer.Header().Get("X-Test"))

	// Write some content
	testContent := []byte("test content")
	n, err := compWriter.Write(testContent)
	require.NoError(t, err)
	assert.Equal(t, len(testContent), n)

	// Close the writer
	err = compWriter.Close()
	require.NoError(t, err)

	// Verify header was passed through
	assert.Equal(t, "test-value", writer.Header().Get("X-Test"))
}
