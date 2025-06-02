package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, s *memStorage)
	}{
		{
			name: "test_counter_operations",
			fn: func(t *testing.T, s *memStorage) {
				// Set counter
				err := s.SetCounter(t.Context(), "test", 10)
				require.NoError(t, err)

				// Increment counter
				err = s.IncrCounter(t.Context(), "test")
				require.NoError(t, err)

				// Get counter map
				counters, err := s.GetMapCounter(t.Context())
				require.NoError(t, err)
				assert.Equal(t, int64(11), counters["test"])
			},
		},
		{
			name: "test_gauge_operations",
			fn: func(t *testing.T, s *memStorage) {
				// Set gauge
				err := s.SetGauge(t.Context(), "test", 10.5)
				require.NoError(t, err)

				// Get gauge map
				gauges, err := s.GetMapGauge(t.Context())
				require.NoError(t, err)
				assert.Equal(t, 10.5, gauges["test"])
			},
		},
		{
			name: "test_concurrent_access",
			fn: func(t *testing.T, s *memStorage) {
				// Simulate concurrent access
				go func() {
					_ = s.SetGauge(t.Context(), "concurrent", 1.0)
				}()
				go func() {
					_ = s.SetCounter(t.Context(), "concurrent", 1)
				}()
				// Check no panic occurs
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := New()
			tt.fn(t, storage)
		})
	}
}
