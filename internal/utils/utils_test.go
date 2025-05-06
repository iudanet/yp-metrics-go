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
			name: "test_andom_numberRange",
			fn: func(t *testing.T) {
				value := GetRandomNumber()
				assert.GreaterOrEqual(t, value, 0.0)
				assert.Less(t, value, 1.0)
			},
		},
		{
			name: "test_random_number_consistency",
			fn: func(t *testing.T) {
				// Поскольку мы используем фиксированный seed,
				// числа должны быть одинаковыми при последовательных вызовах
				first := GetRandomNumber()
				second := GetRandomNumber()
				assert.Equal(t, first, second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}
