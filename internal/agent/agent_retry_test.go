package agent

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/retry"
	"github.com/stretchr/testify/assert"
)

func TestAgentRetryLogic(t *testing.T) {
	tests := []struct {
		name            string
		failTimes       int
		expectedRetries int
		shouldSucceed   bool
	}{
		{
			name:            "successOnFirstAttempt",
			failTimes:       0,
			expectedRetries: 0,
			shouldSucceed:   true,
		},
		{
			name:            "successOnSecondAttempt",
			failTimes:       1,
			expectedRetries: 1,
			shouldSucceed:   true,
		},
		{
			name:            "failAfterAllAttempts",
			failTimes:       3,
			expectedRetries: 3,
			shouldSucceed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				if attempts < tt.failTimes {
					attempts++
					w.WriteHeader(http.StatusInternalServerError)

					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			cfg := &config.AgentConfig{
				PollInterval:     1,
				ReportInterval:   2,
				MetricServerHost: server.URL[7:], // Удаляем "http://" из адреса
			}

			agent := &Agent{
				config: cfg,
				client: &http.Client{Timeout: 1 * time.Second},
			}

			testValue := float64(42.42)
			err := agent.PushGauge("retry_test", testValue)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, retry.ErrMaxRetriesReached))
			}

			assert.Equal(t, tt.failTimes, attempts)
		})
	}
}

func TestRetryWithNetworkError(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval:     2,
		ReportInterval:   10,
		MetricServerHost: "invalid-host:8080",
	}

	agent := &Agent{
		config: cfg,
		client: &http.Client{Timeout: 100 * time.Millisecond},
	}

	err := agent.PushGauge("network_error_test", 42.42)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, retry.ErrMaxRetriesReached))
}
