package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags(args []string) func() {
	oldArgs := os.Args
	os.Args = append([]string{"test"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	return func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	}
}

func TestNewAgentConfig(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedPoll    time.Duration
		expectedReport  time.Duration
		expectedAddress string
		wantErr         bool
	}{
		{
			name:            "default values",
			args:            []string{},
			expectedPoll:    2 * time.Second,
			expectedReport:  10 * time.Second,
			expectedAddress: "localhost:8080",
			wantErr:         false,
		},
		{
			name:            "custom values",
			args:            []string{"-p=5s", "-r=15s", "-a=127.0.0.1:9090"},
			expectedPoll:    5 * time.Second,
			expectedReport:  15 * time.Second,
			expectedAddress: "127.0.0.1:9090",
			wantErr:         false,
		},
		{
			name:    "invalid poll interval",
			args:    []string{"-p=invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer resetFlags(tt.args)()

			cfg := NewAgentConfig()
			err := flag.CommandLine.Parse(tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPoll, cfg.PollInterval)
			assert.Equal(t, tt.expectedReport, cfg.ReportInterval)
			assert.Equal(t, tt.expectedAddress, cfg.MetricServerHost)
		})
	}
}

func TestNewServerConfig(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedAddress string
		wantErr         bool
	}{
		{
			name:            "default value",
			args:            []string{},
			expectedAddress: "localhost:8080",
			wantErr:         false,
		},
		{
			name:            "custom address",
			args:            []string{"-a=127.0.0.1:9090"},
			expectedAddress: "127.0.0.1:9090",
			wantErr:         false,
		},
		{
			name:    "invalid flag",
			args:    []string{"-unknown=value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer resetFlags(tt.args)()

			cfg := NewServerConfig()
			err := flag.CommandLine.Parse(tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.expectedAddress, cfg.MetricServerHost)
		})
	}
}
