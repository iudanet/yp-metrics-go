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
			cfg := config.NewAdminConfig(
				time.Duration(2),
				time.Duration(10),
				server.URL[7:], // Удаляем "http://" из адреса
			)
			store := storage.NewStorage()
			agent := NewAgent(cfg, store)

			tt.fn(t, agent)
		})
	}
}
