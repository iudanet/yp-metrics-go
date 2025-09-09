package config

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentConfig(t *testing.T) {
	cfg, err := ParseAgentFlagsArgs([]string{})
	assert.NoError(t, err)

	assert.Equal(t, 2, cfg.PollInterval, "default poll interval should be 2")
	assert.Equal(t, 10, cfg.ReportInterval, "default report interval should be 10")
	assert.Equal(t, "localhost:8080", cfg.MetricServerHost, "default address should be localhost:8080")
}

func TestParseAgentFlagsArgs(t *testing.T) {
	tests := []struct {
		envVars       map[string]string
		expected      *AgentConfig
		name          string
		args          []string
		expectedError bool
	}{
		{
			name: "defaultValues",
			args: []string{},
			expected: &AgentConfig{
				PollInterval:     2,
				ReportInterval:   10,
				MetricServerHost: "localhost:8080",
				SginKey:          "",
				RateLimit:        1,
			},
			expectedError: false,
		},
		{
			name: "command_line_flags",
			args: []string{"-k", "testKey", "-p", "5", "-r", "15", "-l", "5", "-a", "localhost:9090"},
			expected: &AgentConfig{
				PollInterval:     5,
				ReportInterval:   15,
				MetricServerHost: "localhost:9090",
				SginKey:          "testKey",
				RateLimit:        5,
			},
			expectedError: false,
		},
		{
			name: "env_vars",
			args: []string{},
			envVars: map[string]string{
				"ADDRESS":         "localhost:7070",
				"REPORT_INTERVAL": "20",
				"POLL_INTERVAL":   "3",
				"KEY":             "testKey2",
				"RATE_LIMIT":      "5",
			},
			expected: &AgentConfig{
				PollInterval:     3,
				ReportInterval:   20,
				MetricServerHost: "localhost:7070",
				SginKey:          "testKey2",
				RateLimit:        5,
			},
			expectedError: false,
		},
		{
			name: "env_vars_override_flags",
			args: []string{"-k", "testKey", "-p", "5", "-r", "15", "-l", "5", "-a", "localhost:9090"},
			envVars: map[string]string{
				"ADDRESS":         "localhost:7070",
				"REPORT_INTERVAL": "20",
				"POLL_INTERVAL":   "3",
				"KEY":             "testKey3",
				"RATE_LIMIT":      "6",
			},
			expected: &AgentConfig{
				PollInterval:     3,
				ReportInterval:   20,
				MetricServerHost: "localhost:7070",
				SginKey:          "testKey3",
				RateLimit:        6,
			},
			expectedError: false,
		},
		{
			name:          "invalid_report_interval",
			args:          []string{},
			envVars:       map[string]string{"REPORT_INTERVAL": "invalid"},
			expectedError: true,
		},
		{
			name:          "invalid_poll_interval",
			args:          []string{},
			envVars:       map[string]string{"POLL_INTERVAL": "invalid"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				assert.NoError(t, err)
			}
			// Cleanup env after test
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := ParseAgentFlagsArgs(tt.args)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, cfg)
			}
		})
	}
}

func TestParseAgentFlagsArgs_NegativeValues(t *testing.T) {
	tests := []struct {
		envVars       map[string]string
		name          string
		args          []string
		expectedPoll  int
		expectedError bool
	}{
		{
			name:         "negative_report_interval",
			args:         []string{},
			envVars:      map[string]string{"REPORT_INTERVAL": "-10"},
			expectedPoll: -10,
		},
		{
			name:         "negative_poll_interval",
			args:         []string{},
			envVars:      map[string]string{"POLL_INTERVAL": "-5"},
			expectedPoll: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				assert.NoError(t, err)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := ParseAgentFlagsArgs(tt.args)
			assert.NoError(t, err)

			if tt.name == "negative_report_interval" {
				assert.Equal(t, tt.expectedPoll, cfg.ReportInterval)
			} else if tt.name == "negative_poll_interval" {
				assert.Equal(t, tt.expectedPoll, cfg.PollInterval)
			}
		})
	}
}

func TestSetFromRaw(t *testing.T) {
	cfg := DefaultConfig()

	raw := &rawAgentConfig{
		PollInterval:     5,
		ReportInterval:   15,
		MetricServerHost: "hostfile:9090",
		SginKey:          "signKeyFile",
		RSAPublicKeyPath: "path/to/pubkey",
		RateLimit:        7,
	}
	setFromRaw(cfg, raw, priorityConfigFile)

	assert.Equal(t, 5, cfg.PollInterval.Get())
	assert.Equal(t, 15, cfg.ReportInterval.Get())
	assert.Equal(t, "hostfile:9090", cfg.MetricServerHost.Get())
	assert.Equal(t, "signKeyFile", cfg.SginKey.Get())
	assert.Equal(t, "path/to/pubkey", cfg.RSAPublicKeyPath.Get())
	assert.Equal(t, 7, cfg.RateLimit.Get())

	// Проверка, что при 0/пустых не меняется
	rawEmpty := &rawAgentConfig{}
	setFromRaw(cfg, rawEmpty, priorityConfigFile)
	assert.Equal(t, 5, cfg.PollInterval.Get())
	assert.Equal(t, "hostfile:9090", cfg.MetricServerHost.Get())
}

func TestParseAgentFlagsArgs_Priority(t *testing.T) {
	// Создаём файл конфига с приоритетом ниже, чем env
	const jsonConfig = `{
		"address": "filehost:8080",
		"sgin_key": "filekey",
		"report_interval": 25,
		"poll_interval": 7,
		"rate_limit": 9
	}`
	tmpfile, err := os.CreateTemp("", "cfg-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = io.WriteString(tmpfile, jsonConfig)
	assert.NoError(t, err)
	tmpfile.Close()

	// Передаём флаг config, env и другие флаги, проверяем приоритеты:
	os.Setenv("ADDRESS", "envhost:9090")
	os.Setenv("KEY", "envkey")
	os.Setenv("REPORT_INTERVAL", "30")
	os.Setenv("POLL_INTERVAL", "8")
	os.Setenv("RATE_LIMIT", "10")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("KEY")
		os.Unsetenv("REPORT_INTERVAL")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("RATE_LIMIT")
	}()

	args := []string{"-a", "flaghost:8088", "-k", "flagkey", "-p", "6", "-r", "20", "-l", "8", "-config", tmpfile.Name()}

	cfg, err := ParseAgentFlagsArgs(args)
	assert.NoError(t, err)

	// Env переменные имеют самый высокий приоритет
	assert.Equal(t, "envhost:9090", cfg.MetricServerHost)
	assert.Equal(t, "envkey", cfg.SginKey)
	assert.Equal(t, 30, cfg.ReportInterval)
	assert.Equal(t, 8, cfg.PollInterval)
	assert.Equal(t, 10, cfg.RateLimit)
}

func TestParseAgentFlagsArgs_ConfigFileErrors(t *testing.T) {
	args := []string{"-config", "nonexistentfile.json"}
	_, err := ParseAgentFlagsArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open config file flag")

	os.Setenv("CONFIG_FILE", "nonexistentenv.json")
	defer os.Unsetenv("CONFIG_FILE")

	_, err = ParseAgentFlagsArgs([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open config file env")
}

func TestParseAgentFlagsArgs_InvalidJSON(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "badjson-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.WriteString("{bad json}")
	assert.NoError(t, err)
	tmpfile.Close()

	args := []string{"-config", tmpfile.Name()}
	_, err = ParseAgentFlagsArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal config json error")
}
