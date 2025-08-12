package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAgentWorkers(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T, a *Agent)
	}{
		{
			name: "PollWorker_basic_functionality",
			testFunc: func(t *testing.T, a *Agent) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				go a.PollWorker(ctx)

				// Ждем пока метрики соберутся
				assert.Eventually(t, func() bool {
					gauges, _ := a.reader.GetMapGauge(ctx)
					return len(gauges) > 0
				}, 3*time.Second, 100*time.Millisecond, "Metrics not collected")
			},
		},
		{
			name: "ReportWorkerBatch_basic_functionality",
			testFunc: func(t *testing.T, a *Agent) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				// Инициализируем метрики
				a.GetMetrics(ctx)

				go a.ReportWorkerBatch(ctx)
				time.Sleep(100 * time.Millisecond)
			},
		},
		{
			name: "StartWorkers_and_worker",
			testFunc: func(t *testing.T, a *Agent) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				wg := &sync.WaitGroup{}
				wg.Add(1)
				a.StartWorkers(ctx, wg)
				time.Sleep(100 * time.Millisecond)

				// Добавляем метрики в канал
				metrics := []models.Metrics{
					{ID: "test1", MType: "gauge", Value: ptrFloat64(10.5)},
				}
				a.metricsCh <- metrics
				time.Sleep(100 * time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AgentConfig{
				PollInterval:     1, // 1 second
				ReportInterval:   1,
				MetricServerHost: "localhost:8080",
				RateLimit:        2,
			}
			logger, _ := zap.NewDevelopment()
			store := localStore.New()
			agent := NewAgent(cfg, store, logger)

			tt.testFunc(t, agent)
		})
	}
}

func ptrFloat64(v float64) *float64 { return &v }
