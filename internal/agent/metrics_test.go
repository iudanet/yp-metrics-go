package agent

import (
	"context"
	"runtime"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	mockStorage "github.com/iudanet/yp-metrics-go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMemStatsMapper(t *testing.T) {
	ctx := context.Background()
	store := localStore.New()
	cfg := &config.AgentConfig{}
	logger, _ := zap.NewDevelopment()
	agent := NewAgent(cfg, store, logger)

	// Инициализируем memstats
	runtime.ReadMemStats(agent.memstats)

	err := agent.memStatsMapper(ctx)
	assert.NoError(t, err)

	// Проверяем, что хотя бы некоторые метрики записаны
	gauges, err := store.GetMapGauge(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, gauges)
	assert.Contains(t, gauges, "Alloc")
	assert.Contains(t, gauges, "HeapAlloc")
}

func TestMemStatsMapper_Error(t *testing.T) {
	ctx := context.Background()

	// Create a mock storage that returns errors
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWriter := mockStorage.NewMockMetricWriter(ctrl)

	cfg := &config.AgentConfig{}
	logger, _ := zap.NewDevelopment()

	agent := NewAgent(cfg, nil, logger)
	agent.writer = mockWriter

	// Инициализируем memstats
	runtime.ReadMemStats(agent.memstats)

	// Mock the first SetGauge call to return an error
	mockWriter.EXPECT().SetGauge(gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError).Times(1)

	err := agent.memStatsMapper(ctx)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestCollectPSUtilMetrics(t *testing.T) {
	ctx := context.Background()
	store := localStore.New()
	cfg := &config.AgentConfig{}
	logger, _ := zap.NewDevelopment()
	agent := NewAgent(cfg, store, logger)

	err := agent.collectPSUtilMetrics(ctx)
	assert.NoError(t, err)

	gauges, err := store.GetMapGauge(ctx)
	require.NoError(t, err)
	assert.Contains(t, gauges, "TotalMemory")
	assert.Contains(t, gauges, "FreeMemory")
	// CPU метрики могут быть или нет в зависимости от системы
}

func TestGetMetrics(t *testing.T) {
	ctx := context.Background()

	store := localStore.New()
	store.SetCounter(ctx, "counter1", 10)
	store.SetGauge(ctx, "gauge1", 1.23)

	cfg := &config.AgentConfig{}

	logger, _ := zap.NewDevelopment()
	agent := NewAgent(cfg, store, logger)

	metrics, err := agent.getMetrics(ctx)
	require.NoError(t, err)

	// Должно быть по 1 счетчику и гейджу
	var foundCounter, foundGauge bool
	for _, m := range metrics {
		switch m.MType {
		case models.TypeCounter:
			if m.ID == "counter1" && m.Delta != nil && *m.Delta == 10 {
				foundCounter = true
			}
		case models.TypeGauge:
			if m.ID == "gauge1" && m.Value != nil && *m.Value == 1.23 {
				foundGauge = true
			}
		}
	}

	assert.True(t, foundCounter, "Counter metric должен присутствовать")
	assert.True(t, foundGauge, "Gauge metric должен присутствовать")
}
