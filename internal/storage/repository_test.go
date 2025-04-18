package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, s Repository)
	}{
		{
			name: "test counter operations",
			fn: func(t *testing.T, s Repository) {
				// Set counter
				err := s.SetCounter("test", 10)
				require.NoError(t, err)

				// Increment counter
				err = s.IncrCounter("test")
				require.NoError(t, err)

				// Get counter map
				counters, err := s.GetMapCounter()
				require.NoError(t, err)
				assert.Equal(t, int64(11), counters["test"])
			},
		},
		{
			name: "test gauge operations",
			fn: func(t *testing.T, s Repository) {
				// Set gauge
				err := s.SetGauge("test", 10.5)
				require.NoError(t, err)

				// Get gauge map
				gauges, err := s.GetMapGauge()
				require.NoError(t, err)
				assert.Equal(t, 10.5, gauges["test"])
			},
		},
		{
			name: "test concurrent access",
			fn: func(t *testing.T, s Repository) {
				// Simulate concurrent access
				go func() {
					_ = s.SetGauge("concurrent", 1.0)
				}()
				go func() {
					_ = s.SetCounter("concurrent", 1)
				}()
				// Check no panic occurs
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.fn(t, storage)
		})
	}
}
