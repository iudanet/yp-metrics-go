package agent

import (
	"context"
	"net/http"
	"runtime"
	"sync"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/iudanet/yp-metrics-go/internal/utils"
	"go.uber.org/zap"
)

type Agent struct {
	memstats  *runtime.MemStats
	config    *config.AgentConfig
	writer    storage.MetricWriter
	counter   storage.CounterIncrementer
	reader    storage.MetricReader
	client    *http.Client
	logger    *zap.Logger
	metricsCh chan []models.Metrics
	workerWg  sync.WaitGroup
}

func NewAgent(cfg *config.AgentConfig, storage storage.Repository, logger *zap.Logger) *Agent {
	agent := &Agent{
		memstats:  &runtime.MemStats{},
		config:    cfg,
		writer:    storage,
		counter:   storage,
		reader:    storage,
		client:    &http.Client{},
		metricsCh: make(chan []models.Metrics, cfg.RateLimit),

		logger: logger,
	}
	return agent
}

func (a *Agent) GetMetrics(ctx context.Context) error {
	// увеличиваем каждую иттерацию
	err := a.counter.IncrCounter(ctx, "PollCount")
	if err != nil {
		return err
	}
	// получаем статистику памяти
	a.getMemStats(ctx)
	// получаем рандомное число
	err = a.writer.SetGauge(ctx, "RandomValue", utils.GetRandomNumber())
	if err != nil {
		return err
	}

	err = a.collectPSUtilMetrics(ctx)
	if err != nil {
		a.logger.Error("Failed to collect psutil metrics", zap.Error(err))
	}

	return nil
}
