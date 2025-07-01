package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLogging(t *testing.T) {
	// Setup logger with observer
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core) // используем *zap.Logger вместо SugaredLogger

	// Setup service
	cfg := config.NewServerConfig()
	store := localStore.New()
	svc := &service{
		storage: store,
		viewer:  store,
		config:  cfg,
		logger:  logger, // передаем *zap.Logger
	}

	// Test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/not-found" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})
	loggingHandler := svc.WithLogging(handler)

	tests := []struct {
		name       string
		method     string
		path       string
		status     int
		bodySize   int
		expectLogs bool
	}{
		{
			name:       "Successful_request",
			method:     "GET",
			path:       "/test",
			status:     http.StatusOK,
			bodySize:   13, // "test response" has 13 bytes
			expectLogs: true,
		},
		{
			name:       "Not_found",
			method:     "GET",
			path:       "/not-found",
			status:     http.StatusNotFound,
			bodySize:   0,
			expectLogs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			loggingHandler.ServeHTTP(rr, req)

			if tt.expectLogs {
				// Wait a moment for logs to be written
				time.Sleep(100 * time.Millisecond)

				logs := recorded.All()
				assert.GreaterOrEqual(t, len(logs), 1, "Expected at least one log entry")

				if len(logs) > 0 {
					logEntry := logs[len(logs)-1]
					assert.Equal(t, "request", logEntry.Message)

					fields := logEntry.ContextMap()
					assert.Equal(t, tt.path, fields["uri"])
					assert.Equal(t, tt.method, fields["method"])
					assert.EqualValues(t, tt.status, fields["status"])
					assert.EqualValues(t, tt.bodySize, fields["size"])
					assert.NotNil(t, fields["duration"])
				}
			}
		})
	}
}

func TestWithLogging_Panics(t *testing.T) {
	// Setup logger with observer
	core, _ := observer.New(zap.InfoLevel)
	logger := zap.New(core) // используем *zap.Logger

	// Setup service
	cfg := config.NewServerConfig()
	store := localStore.New()
	svc := &service{
		storage: store,
		viewer:  store,
		config:  cfg,
		logger:  logger,
	}

	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	loggingHandler := svc.WithLogging(panicHandler)

	req := httptest.NewRequest("GET", "/panic", nil)
	rr := httptest.NewRecorder()

	assert.Panics(t, func() {
		loggingHandler.ServeHTTP(rr, req)
	}, "Middleware should not recover panics")
}

func TestLoggingResponseWriter(t *testing.T) {
	// Setup
	rr := httptest.NewRecorder()
	data := &responseData{}
	lrw := loggingResponseWriter{
		ResponseWriter: rr,
		responseData:   data,
	}

	// Test WriteHeader
	lrw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, data.status)

	// Test Write
	n, err := lrw.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, 4, data.size)

	// Test multiple writes
	n, err = lrw.Write([]byte(" content"))
	assert.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, 12, data.size)
}
