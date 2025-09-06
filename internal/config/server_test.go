package config

import (
	"io"
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

func TestSetFromRawServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	raw := &rawServerConfig{
		Address:       "rawhost:9090",
		SginKey:       "rawsign",
		CryptoKey:     "rawprivate.pem",
		StoreFile:     "/tmp/raw.json",
		DatabaseDSN:   "postgres://user:pass@localhost/db",
		StoreInterval: "60s",
		Restore:       true,
	}

	err := setFromRawServerConfig(cfg, raw, priorityConfigFile)
	assert.NoError(t, err)

	assert.Equal(t, "rawhost:9090", cfg.MetricServerHost.Get())
	assert.Equal(t, "rawsign", cfg.SginKey.Get())
	assert.Equal(t, "rawprivate.pem", cfg.RSAPrivateKeyPath.Get())
	assert.Equal(t, "/tmp/raw.json", cfg.StoragePath.Get())
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.StorageDatabaseDSN.Get())
	assert.Equal(t, 60, cfg.StorageStoreInterval.Get())
	assert.True(t, cfg.StorageRestore.Get())

	// Проверка ошибочного формата store_interval
	rawInvalid := &rawServerConfig{StoreInterval: "invalid"}
	err = setFromRawServerConfig(cfg, rawInvalid, priorityConfigFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid store_interval")
}
func TestParseServerFlagsArgs_Priority(t *testing.T) {
	const rawConfig = `{
		"address": "filehost:8080",
		"sgin_key": "filekey",
		"crypto_key": "filekey.pem",
		"store_file": "/file/path.json",
		"database_dsn": "postgres://file/db",
		"store_interval": "45s",
		"restore": true
	}`
	tmpfile, err := os.CreateTemp("", "srvconfig-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = io.WriteString(tmpfile, rawConfig)
	assert.NoError(t, err)
	tmpfile.Close()

	os.Setenv("ADDRESS", "envhost:9090")
	os.Setenv("KEY", "envkey")
	os.Setenv("CRYPTO_KEY", "envprivate.pem")
	os.Setenv("FILE_STORAGE_PATH", "/env/path.json")
	os.Setenv("DATABASE_DSN", "postgres://env/db")
	os.Setenv("STORE_INTERVAL", "50")
	os.Setenv("RESTORE", "false")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("KEY")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("RESTORE")
	}()

	args := []string{"-a", "flaghost:8088", "-k", "flagkey", "-f", "/flag/path.json", "-d", "postgres://flag/db", "-i", "40", "-r=true", "-config", tmpfile.Name()}
	cfg, err := ParseServerFlagsArgs(args)
	assert.NoError(t, err)

	// Env имеет самый высокий приоритет, override все
	assert.Equal(t, "envhost:9090", cfg.MetricServerHost)
	assert.Equal(t, "envkey", cfg.SginKey)
	assert.Equal(t, "envprivate.pem", cfg.RSAPrivateKeyPath)
	assert.Equal(t, "/env/path.json", cfg.Storage.Path)
	assert.Equal(t, "postgres://env/db", cfg.Storage.DatabaseDSN)
	assert.Equal(t, 50, cfg.Storage.StoreInterval)
	assert.False(t, cfg.Storage.Restore)
}

func TestParseServerFlagsArgs_ConfigFileErrors(t *testing.T) {
	args := []string{"-config", "nonexistentfile.json"}
	_, err := ParseServerFlagsArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open config file flag")

	os.Setenv("CONFIG_FILE", "nonexistentenv.json")
	defer os.Unsetenv("CONFIG_FILE")

	_, err = ParseServerFlagsArgs([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open config file env")
}

func TestParseServerFlagsArgs_InvalidJSON(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "badjson-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.WriteString("{bad json}")
	assert.NoError(t, err)
	tmpfile.Close()

	args := []string{"-config", tmpfile.Name()}
	_, err = ParseServerFlagsArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal config json error")
}

func TestParseServerFlagsArgs_InvalidEnvironment(t *testing.T) {
	// Проверка ошибки для STORE_INTERVAL
	os.Setenv("STORE_INTERVAL", "invalid")
	defer os.Unsetenv("STORE_INTERVAL")

	_, err := ParseServerFlagsArgs([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid STORE_INTERVAL env")

	// Чтобы не мешались переменные окружения:
	os.Unsetenv("STORE_INTERVAL")

	// Проверка ошибки для RESTORE
	os.Setenv("RESTORE", "invalid")
	defer os.Unsetenv("RESTORE")

	_, err = ParseServerFlagsArgs([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid RESTORE env")
}
