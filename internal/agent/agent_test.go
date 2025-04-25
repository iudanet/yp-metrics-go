package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestAgentConfig struct {
	PollInterval     time.Duration
	ReportInterval   time.Duration
	MetricServerHost string
}

func (c *TestAgentConfig) GetPollInterval() time.Duration {
	return c.PollInterval
}

func (c *TestAgentConfig) GetReportInterval() time.Duration {
	return c.ReportInterval
}

func (c *TestAgentConfig) GetMetricServerHost() string {
	return c.MetricServerHost
}

func TestAgent(t *testing.T) {
	// Создаем тестовый HTTP сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name string
		fn   func(t *testing.T, a *Agent)
	}{
		{
			name: "test metrics collection",
			fn: func(t *testing.T, a *Agent) {
				a.GetMetrics()

				gauges, err := a.storage.GetMapGauge()
				require.NoError(t, err)
				assert.NotEmpty(t, gauges)

				counters, err := a.storage.GetMapCounter()
				require.NoError(t, err)
				assert.Equal(t, int64(1), counters["PollCount"])
			},
		},
		{
			name: "test push counter",
			fn: func(t *testing.T, a *Agent) {
				err := a.PushCounter("test", 10)
				assert.NoError(t, err)
			},
		},
		{
			name: "test push gauge",
			fn: func(t *testing.T, a *Agent) {
				err := a.PushGouge("test", 10.5)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AgentConfig{
				PollInterval:     2 * time.Second,
				ReportInterval:   10 * time.Second,
				MetricServerHost: server.URL[7:], // Удаляем "http://" из адреса
			}
			serverHost := server.URL[7:]
			cfg.MetricServerHost = serverHost // Удаляем "http://" из адреса
			store := storage.NewStorage()
			agent := NewAgent(cfg, store)

			tt.fn(t, agent)
		})
	}
}
