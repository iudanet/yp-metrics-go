package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentConfig(t *testing.T) {
	cfg := NewAgentConfig()

	assert.Equal(t, 2, cfg.PollInterval, "default poll interval should be 2")
	assert.Equal(t, 10, cfg.ReportInterval, "default report interval should be 10")
	assert.Equal(t, "localhost:8080", cfg.MetricServerHost, "default address should be localhost:8080")
}
