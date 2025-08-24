package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomNumber(t *testing.T) {
	tests := []struct {
		fn   func(t *testing.T)
		name string
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

func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
		data     []byte
	}{
		{
			name:     "Empty_data_and_key",
			data:     []byte(""),
			key:      "",
			expected: "",
		},
		{
			name:     "Empty_data_with_key",
			data:     []byte(""),
			key:      "secret",
			expected: hex.EncodeToString(hmac.New(sha256.New, []byte("secret")).Sum(nil)),
		},
		{
			name:     "Simple_data",
			data:     []byte("test"),
			key:      "secret",
			expected: "0329a06b62cd16b33eb6792be8c60b158d89a2ee3a876fce9a881ebb488c0914",
		},
		{
			name:     "Different_keys_same_data",
			data:     []byte("test"),
			key:      "secret1",
			expected: "e025df0a6771f7b2082ce6f92de78702f49235f3453821b6b710eadb831d1249",
		},
		{
			name:     "Special_characters",
			data:     []byte("!@#$%^&*()_+"),
			key:      "key!@#",
			expected: "5d1f1213ff5903f041d39d0ee155bbc23c2bdf972a7a8de138a0d771f1a03187",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateHash(tt.data, tt.key)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else if tt.key != "" {
				// Проверяем что хеш соответствует HMAC-SHA256
				h := hmac.New(sha256.New, []byte(tt.key))
				h.Write(tt.data)
				expected := hex.EncodeToString(h.Sum(nil))
				assert.Equal(t, expected, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}

	t.Run("Consistency", func(t *testing.T) {
		data := []byte("consistent data")
		key := "secret"
		hash1 := CalculateHash(data, key)
		hash2 := CalculateHash(data, key)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("Different data same key", func(t *testing.T) {
		key := "secret"
		hash1 := CalculateHash([]byte("data1"), key)
		hash2 := CalculateHash([]byte("data2"), key)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("Same data different keys", func(t *testing.T) {
		data := []byte("same data")
		hash1 := CalculateHash(data, "key1")
		hash2 := CalculateHash(data, "key2")
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("Large data", func(t *testing.T) {
		data := make([]byte, 10*1024*1024) // 10 MB
		for i := range data {
			data[i] = byte(rand.IntN(256))
		}
		key := "large-data-key"

		// Просто проверяем что не падает
		result := CalculateHash(data, key)
		assert.NotEmpty(t, result)
		assert.Len(t, result, 64) // SHA256 всегда 64 hex символа
	})
}
