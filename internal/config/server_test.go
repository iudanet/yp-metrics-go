package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseServerFlags(t *testing.T) {
	tests := []struct {
		envVars  map[string]string
		expected *ServerConfig
		name     string
		args     []string
	}{
		{
			name: "defaultValues",
			args: []string{"cmd"},
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
			args: []string{"cmd", "-a", "127.0.0.1:9090", "-f", "/tmp/custom.json", "-d", "postgres://user:pass@localhost/db", "-k", "secret", "-i", "60", "-r=true"},
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
			args: []string{"cmd"},
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
			if len(tt.args) == 0 {
				t.Fatal("tt.args cannot be empty")
			}
			// Сбрасываем флаги перед каждым тестом
			flag.CommandLine = flag.NewFlagSet(tt.args[0], flag.ExitOnError)
			// Сохраняем оригинальные значения
			oldArgs := os.Args
			oldEnv := make(map[string]string)
			for k := range tt.envVars {
				oldEnv[k] = os.Getenv(k)
			}

			defer func() {
				// Восстанавливаем окружение
				os.Args = oldArgs
				for k, v := range oldEnv {
					os.Setenv(k, v)
				}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			}()

			// Устанавливаем тестовые значения
			os.Args = tt.args
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Выполняем тестируемую функцию
			cfg := ParseServerFlags()

			// Проверяем результаты
			assert.Equal(t, tt.expected.MetricServerHost, cfg.MetricServerHost)
			assert.Equal(t, tt.expected.Storage, cfg.Storage)
			assert.Equal(t, tt.expected.SginKey, cfg.SginKey)
		})
	}
}
