package config

import (
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
