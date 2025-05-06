package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentConfig(t *testing.T) {
	cfg := NewAgentConfig()

	assert.Equal(t, 2, cfg.PollInterval, "default poll interval should be 2")
	assert.Equal(t, 10, cfg.ReportInterval, "default report interval should be 10")
	assert.Equal(t, "localhost:8080", cfg.MetricServerHost, "default address should be localhost:8080")
}

func TestParseAgentFlags(t *testing.T) {
	// Сохраняем оригинальные аргументы и флаги
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
	}()

	programName := "agent"

	tests := []struct {
		name          string
		args          []string
		envVars       map[string]string
		expected      *AgentConfig
		expectedError bool
	}{
		{
			name: "default_values",
			args: []string{programName},
			expected: &AgentConfig{
				PollInterval:     2,
				ReportInterval:   10,
				MetricServerHost: "localhost:8080",
			},
		},
		{
			name: "command_line_flags",
			args: []string{programName, "-p", "5", "-r", "15", "-a", "localhost:9090"},
			expected: &AgentConfig{
				PollInterval:     5,
				ReportInterval:   15,
				MetricServerHost: "localhost:9090",
			},
		},
		{
			name: "env_vars",
			args: []string{programName},
			envVars: map[string]string{
				"ADDRESS":         "localhost:7070",
				"REPORT_INTERVAL": "20",
				"POLL_INTERVAL":   "3",
			},
			expected: &AgentConfig{
				PollInterval:     3,
				ReportInterval:   20,
				MetricServerHost: "localhost:7070",
			},
		},
		{
			name: "env_vars_override_flags",
			args: []string{programName, "-p", "5", "-r", "15", "-a", "localhost:9090"},
			envVars: map[string]string{
				"ADDRESS":         "localhost:7070",
				"REPORT_INTERVAL": "20",
				"POLL_INTERVAL":   "3",
			},
			expected: &AgentConfig{
				PollInterval:     3,
				ReportInterval:   20,
				MetricServerHost: "localhost:7070",
			},
		},
		{
			name: "invalid_report_interval",
			args: []string{programName},
			envVars: map[string]string{
				"REPORT_INTERVAL": "invalid",
			},
			expectedError: true,
		},
		{
			name: "invalid_poll_interval",
			args: []string{programName},
			envVars: map[string]string{
				"POLL_INTERVAL": "invalid",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем флаги перед каждым тестом
			flag.CommandLine = flag.NewFlagSet(programName, flag.ExitOnError)

			// Очищаем переменные окружения
			os.Unsetenv("ADDRESS")
			os.Unsetenv("REPORT_INTERVAL")
			os.Unsetenv("POLL_INTERVAL")

			// Устанавливаем тестовые переменные окружения
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				assert.NoError(t, err, "failed to set env variable")
			}

			// Устанавливаем тестовые аргументы командной строки
			os.Args = tt.args

			// Выполняем тестируемую функцию
			cfg, err := ParseAgentFlags()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, cfg)
			}
		})
	}
}

func TestParseAgentFlags_NegativeValues(t *testing.T) {
	// Сохраняем оригинальные значения
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine

	// Очищаем все переменные окружения перед тестом
	os.Unsetenv("ADDRESS")
	os.Unsetenv("REPORT_INTERVAL")
	os.Unsetenv("POLL_INTERVAL")

	defer func() {
		// Восстанавливаем оригинальные значения
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
		// Очищаем переменные окружения после теста
		os.Unsetenv("ADDRESS")
		os.Unsetenv("REPORT_INTERVAL")
		os.Unsetenv("POLL_INTERVAL")
	}()

	t.Run("negative report interval", func(t *testing.T) {
		// Сбрасываем флаги
		flag.CommandLine = flag.NewFlagSet("agent", flag.ExitOnError)
		os.Args = []string{"agent"}

		err := os.Setenv("REPORT_INTERVAL", "-10")
		assert.NoError(t, err)

		cfg, err := ParseAgentFlags()
		assert.NoError(t, err)
		assert.Equal(t, -10, cfg.ReportInterval)
	})

	t.Run("negative poll interval", func(t *testing.T) {
		// Сбрасываем флаги
		flag.CommandLine = flag.NewFlagSet("agent", flag.ExitOnError)
		os.Args = []string{"agent"}

		// Очищаем предыдущее значение
		os.Unsetenv("REPORT_INTERVAL")

		err := os.Setenv("POLL_INTERVAL", "-5")
		assert.NoError(t, err)

		cfg, err := ParseAgentFlags()
		assert.NoError(t, err)
		assert.Equal(t, -5, cfg.PollInterval)
	})
}
