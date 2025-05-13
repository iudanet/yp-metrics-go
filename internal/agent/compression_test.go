package agent

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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