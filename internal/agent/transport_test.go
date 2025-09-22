package agent

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/retry"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAgentRetryLogic(t *testing.T) {
	tests := []struct {
		name          string
		failTimes     int
		shouldSucceed bool
	}{
		{
			name:          "successOnFirstAttempt",
			failTimes:     0,
			shouldSucceed: true,
		},
		{
			name:          "successOnSecondAttempt",
			failTimes:     1,
			shouldSucceed: true,
		},
		{
			name:          "failAfterAllAttempts",
			failTimes:     3,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				if attempts < tt.failTimes {
					attempts++
					w.WriteHeader(http.StatusInternalServerError)

					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			cfg := &config.AgentConfig{
				PollInterval:     1,
				ReportInterval:   2,
				MetricServerHost: server.URL[7:], // Удаляем "http://" из адреса
			}

			store := localStore.New()
			logger, _ := zap.NewDevelopment()
			agent := NewAgent(cfg, store, logger)
			agent.client = &http.Client{Timeout: 1 * time.Second}

			testValue := float64(42.42)
			err := agent.PushGauge("retry_test", testValue)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, retry.ErrMaxRetriesReached))
			}

			assert.Equal(t, tt.failTimes, attempts)
		})
	}
}

func TestRetryWithNetworkError(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval:     2,
		ReportInterval:   10,
		MetricServerHost: "invalid-host:8080",
	}

	store := localStore.New()
	logger, _ := zap.NewDevelopment()
	agent := NewAgent(cfg, store, logger)
	agent.client = &http.Client{Timeout: 100 * time.Millisecond}

	err := agent.PushGauge("network_error_test", 42.42)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, retry.ErrMaxRetriesReached))
}

func TestCompressData(t *testing.T) {
	testCases := []struct {
		name        string
		inputData   []byte
		expectError bool
	}{
		{
			name:        "empty_data",
			inputData:   []byte{},
			expectError: false,
		},
		{
			name:        "simple_string",
			inputData:   []byte("test string for compression"),
			expectError: false,
		},
		{
			name:        "json_data",
			inputData:   []byte(`{"id":"testMetric","type":"gauge","value":42.0}`),
			expectError: false,
		},
		{
			name:        "large_data",
			inputData:   bytes.Repeat([]byte("large data test "), 1000),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Compress the data
			compressed, err := compressData(tc.inputData)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, compressed)

			// The compressed data should be different from the input
			if len(tc.inputData) > 0 {
				assert.NotEqual(t, tc.inputData, compressed)
			}

			// Decompress the data to verify it matches the original
			reader, err := gzip.NewReader(bytes.NewReader(compressed))
			require.NoError(t, err)

			decompressed, err := io.ReadAll(reader)
			require.NoError(t, err)

			// Verify the decompressed data matches the original input
			assert.Equal(t, tc.inputData, decompressed)

			err = reader.Close()
			require.NoError(t, err)
		})
	}
}

func TestCompressDecompressRoundTrip(t *testing.T) {
	originalData := []byte(`{"id":"MetricName","type":"counter","delta":12345}`)

	// Compress
	compressed, err := compressData(originalData)
	require.NoError(t, err)

	// The compressed data size may vary and could be larger for small inputs due to gzip headers
	// We're just verifying the compression/decompression process works correctly

	// Decompress
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Verify round trip
	assert.Equal(t, originalData, decompressed)
}
