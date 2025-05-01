package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerConfig(t *testing.T) {
	cfg := NewServerConfig()

	assert.Equal(t, "localhost:8080", cfg.MetricServerHost, "default address should be localhost:8080")
}

func TestParseServerFlags_Environment(t *testing.T) {
	// Сохраняем оригинальное значение переменной окружения
	oldAddress := os.Getenv("ADDRESS")

	// Восстанавливаем оригинальное значение после теста
	defer func() {
		os.Setenv("ADDRESS", oldAddress)
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected ServerConfig
	}{
		{
			name: "env_overrides_default",
			envVars: map[string]string{
				"ADDRESS": "localhost:9090",
			},
			expected: ServerConfig{
				MetricServerHost: "localhost:9090",
			},
		},
		{
			name:    "no_env_vars",
			envVars: map[string]string{},
			expected: ServerConfig{
				MetricServerHost: "localhost:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем переменную окружения
			os.Unsetenv("ADDRESS")

			// Устанавливаем тестовые переменные окружения
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				assert.NoError(t, err, "failed to set env variable")
			}

			cfg := NewServerConfig()
			if addr := os.Getenv("ADDRESS"); addr != "" {
				cfg.MetricServerHost = addr
			}

			assert.Equal(t, tt.expected, *cfg)
		})
	}
}
