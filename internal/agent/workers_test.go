package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	mockStorage "github.com/iudanet/yp-metrics-go/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAgentWorkers(t *testing.T) {
	tests := []struct {
		testFunc func(t *testing.T, a *Agent)
		name     string
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

func TestReportWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := mockStorage.NewMockTransport(ctrl)
	mockReader := mockStorage.NewMockMetricReader(ctrl)

	cfg := &config.AgentConfig{
		PollInterval:     1,
		ReportInterval:   1, // 1 second interval
		MetricServerHost: "localhost:8080",
		RateLimit:        1,
	}
	logger, _ := zap.NewDevelopment()
	store := localStore.New()

	agent := NewAgent(cfg, store, logger)
	agent.transport = mockTransport
	agent.reader = mockReader

	// Mock данные для счетчиков и гаугов
	counters := map[string]int64{"testCounter": 42}
	gauges := map[string]float64{"testGauge": 3.14}

	// Устанавливаем ожидания с использованием каналов для синхронизации
	counterCalled := make(chan bool, 1)
	gaugeCalled := make(chan bool, 1)
	transportCalled := make(chan bool, 2)

	mockReader.EXPECT().GetMapCounter(gomock.Any()).DoAndReturn(func(ctx context.Context) (map[string]int64, error) {
		counterCalled <- true
		return counters, nil
	}).MinTimes(1)

	mockReader.EXPECT().GetMapGauge(gomock.Any()).DoAndReturn(func(ctx context.Context) (map[string]float64, error) {
		gaugeCalled <- true
		return gauges, nil
	}).MinTimes(1)

	mockTransport.EXPECT().PushMetric(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metric models.Metrics) error {
		transportCalled <- true
		return nil
	}).MinTimes(2)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Запускаем ReportWorker
	go agent.ReportWorker(ctx)

	// Ждем вызовов методов
	select {
	case <-counterCalled:
		// GetMapCounter вызван
	case <-time.After(2 * time.Second):
		t.Fatal("GetMapCounter not called within timeout")
	}

	select {
	case <-gaugeCalled:
		// GetMapGauge вызван
	case <-time.After(2 * time.Second):
		t.Fatal("GetMapGauge not called within timeout")
	}

	// Ждем хотя бы одного вызова PushMetric
	select {
	case <-transportCalled:
		// PushMetric вызван
	case <-time.After(2 * time.Second):
		t.Fatal("PushMetric not called within timeout")
	}

	// Отменяем контекст для остановки воркера
	cancel()
}

func ptrFloat64(v float64) *float64 { return &v }
