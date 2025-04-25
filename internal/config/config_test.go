package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentConfig(t *testing.T) {
	cfg := NewAgentConfig()

	assert.Equal(t, 2*time.Second, cfg.PollInterval, "default poll interval should be 2s")
	assert.Equal(t, 10*time.Second, cfg.ReportInterval, "default report interval should be 10s")
	assert.Equal(t, "localhost:8080", cfg.MetricServerHost, "default address should be localhost:8080")
}

func TestNewServerConfig(t *testing.T) {
	cfg := NewServerConfig()

	assert.Equal(t, "localhost:8080", cfg.MetricServerHost, "default address should be localhost:8080")
}

func TestParseAgentFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	tests := []struct {
		name     string
		args     []string
		expected AgentConfig
	}{
		{
			name: "default",
			args: []string{"test"},
			expected: AgentConfig{
				PollInterval:     2 * time.Second,
				ReportInterval:   10 * time.Second,
				MetricServerHost: "localhost:8080",
			},
		},
		{
			name: "custom values",
			args: []string{
				"test",
				"-p", "5",
				"-r", "15",
				"-a", "127.0.0.1:9090",
			},
			expected: AgentConfig{
				PollInterval:     5 * time.Second,
				ReportInterval:   15 * time.Second,
				MetricServerHost: "127.0.0.1:9090",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем тестовые аргументы
			os.Args = tt.args
			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg := ParseAgentFlags()
			assert.Equal(t, tt.expected, *cfg)
		})
	}
}

func TestParseServerFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	tests := []struct {
		name     string
		args     []string
		expected ServerConfig
	}{
		{
			name: "default",
			args: []string{"test"},
			expected: ServerConfig{
				MetricServerHost: "localhost:8080",
			},
		},
		{
			name: "custom address",
			args: []string{"test", "-a", "127.0.0.1:9090"},
			expected: ServerConfig{
				MetricServerHost: "127.0.0.1:9090",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg := ParseServerFlags()
			assert.Equal(t, tt.expected, *cfg)
		})
	}
}
