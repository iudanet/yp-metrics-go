package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseServerFlagsArgs(t *testing.T) {
	tests := []struct {
		envVars  map[string]string
		expected *ServerConfig
		name     string
		args     []string
	}{
		{
			name: "defaultValues",
			args: []string{},
			expected: &ServerConfig{
				MetricServerHost: "localhost:8080",
				Storage: Storage{
					Path:          "/tmp/metrics-db.json",
					StoreInterval: 300,
					Restore:       false,
					DatabaseDSN:   "",
				},
			},
		},
		{
			name: "Command_line_flags",
			args: []string{"-a", "127.0.0.1:9090", "-f", "/tmp/custom.json", "-d", "postgres://user:pass@localhost/db", "-k", "secret", "-i", "60", "-r=true"},
			expected: &ServerConfig{
				MetricServerHost: "127.0.0.1:9090",
				Storage: Storage{
					Path:          "/tmp/custom.json",
					StoreInterval: 60,
					Restore:       true,
					DatabaseDSN:   "postgres://user:pass@localhost/db",
				},
				SginKey: "secret",
			},
		},
		{
			name: "Environment_variables",
			args: []string{},
			envVars: map[string]string{
				"ADDRESS":           "0.0.0.0:8080",
				"FILE_STORAGE_PATH": "/env/path.json",
				"DATABASE_DSN":      "postgres://env@localhost/db",
				"KEY":               "env-secret",
				"STORE_INTERVAL":    "120",
				"RESTORE":           "true",
			},
			expected: &ServerConfig{
				MetricServerHost: "0.0.0.0:8080",
				Storage: Storage{
					Restore:       true,
					Path:          "/env/path.json",
					StoreInterval: 120,
					DatabaseDSN:   "postgres://env@localhost/db",
				},
				SginKey: "env-secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем флаги перед каждым тестом
			// flag.CommandLine = flag.NewFlagSet("test", flag.ExitOnError)

			// Устанавливаем переменные окружения
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				assert.NoError(t, err)
			}

			// Очищаем переменные окружения после теста
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := ParseServerFlagsArgs(tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg)
		})
	}
}
