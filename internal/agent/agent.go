// Package agent implements the metrics collection agent functionality.
// It provides the core agent implementation for gathering system metrics and reporting them to the server.
package agent

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"sync"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/iudanet/yp-metrics-go/internal/utils"
	"go.uber.org/zap"
)

// Agent представляет агент для сбора и отправки метрик
type Agent struct {
	writer    storage.MetricWriter
	counter   storage.CounterIncrementer
	reader    storage.MetricReader
	memstats  *runtime.MemStats
	config    *config.AgentConfig
	client    *http.Client
	logger    *zap.Logger
	metricsCh chan []models.Metrics
	ip        net.IP
	workerWg  sync.WaitGroup
}

// NewAgent создает новый экземпляр Agent
func NewAgent(cfg *config.AgentConfig, storage storage.Repository, logger *zap.Logger) *Agent {
	localIP, err := getDefaultInterfaceIP()
	if err != nil {
		logger.Error("Failed to get local IP", zap.Error(err))
		localIP = net.IPv4zero
	}
	logger.Info("Local IP", zap.String("ip", localIP.String()))
	agent := &Agent{
		memstats:  &runtime.MemStats{},
		config:    cfg,
		writer:    storage,
		counter:   storage,
		reader:    storage,
		client:    &http.Client{},
		metricsCh: make(chan []models.Metrics, cfg.RateLimit),

		logger: logger,
		ip:     localIP,
	}
	return agent
}

// GetMetrics собирает метрики с системы
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
