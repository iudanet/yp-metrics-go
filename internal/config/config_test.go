package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAdminConfig(t *testing.T) {
	tests := []struct {
		name           string
		pollInterval   time.Duration
		reportInterval time.Duration
		serverHost     string
		want           *AdminConfig
	}{
		{
			name:           "basic config",
			pollInterval:   time.Duration(2),
			reportInterval: time.Duration(10),
			serverHost:     "localhost:8080",
			want: &AdminConfig{
				PollInterval:     time.Duration(2),
				ReportInterval:   time.Duration(10),
				MetricServerHost: "localhost:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAdminConfig(tt.pollInterval, tt.reportInterval, tt.serverHost)
			assert.Equal(t, tt.want, got)
		})
	}
}
