package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomNumber(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "test_random_number_range",
			fn: func(t *testing.T) {
				value := GetRandomNumber()
				assert.GreaterOrEqual(t, value, 0.0)
				assert.Less(t, value, 1.0)
			},
		},
		{
			name: "test_random_number_different_values",
			fn: func(t *testing.T) {
				values := make(map[float64]struct{})
				for range 1000 {
					v := GetRandomNumber()
					if _, exists := values[v]; exists {
						t.Fatalf("Duplicate random value: %v", v)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}
