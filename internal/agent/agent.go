package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/iudanet/yp-metrics-go/internal/utils"
	"go.uber.org/zap"
)

type Agent struct {
	memstats *runtime.MemStats
	config   *config.AgentConfig
	// storage  storage.Repository
	writer          storage.MetricWriter
	counter         storage.CounterIncrementer
	reader          storage.MetricReader
	client          *http.Client
	backoffSchedule []time.Duration
	logger          *zap.Logger
}

func NewAgent(cfg *config.AgentConfig, storage storage.Repository, logger *zap.Logger) *Agent {
	agent := &Agent{
		memstats: &runtime.MemStats{},
		config:   cfg,
		writer:   storage,
		counter:  storage,
		reader:   storage,
		client:   &http.Client{},
		backoffSchedule: []time.Duration{
			1 * time.Second,
			3 * time.Second,
			5 * time.Second,
		},
		logger: logger,
	}
	return agent
}

func (a *Agent) GetMetrics(ctx context.Context) {
	// увеличиваем каждую иттерацию
	a.counter.IncrCounter(ctx, "PollCount")
	// получаем статистику памяти
	a.getMemStats(ctx)
	// получаем рандомное число
	a.writer.SetGauge(ctx, "RandomValue", utils.GetRandomNumber())
}

func (a *Agent) getMemStats(ctx context.Context) {
	runtime.ReadMemStats(a.memstats)
	a.memStatsMapper(ctx)
}

func (a *Agent) memStatsMapper(ctx context.Context) {
	a.writer.SetGauge(ctx, "Alloc", float64(a.memstats.Alloc))
	a.writer.SetGauge(ctx, "BuckHashSys", float64(a.memstats.BuckHashSys))
	a.writer.SetGauge(ctx, "Frees", float64(a.memstats.Frees))
	a.writer.SetGauge(ctx, "GCCPUFraction", a.memstats.GCCPUFraction)
	a.writer.SetGauge(ctx, "GCSys", float64(a.memstats.GCSys))
	a.writer.SetGauge(ctx, "HeapAlloc", float64(a.memstats.HeapAlloc))
	a.writer.SetGauge(ctx, "HeapIdle", float64(a.memstats.HeapIdle))
	a.writer.SetGauge(ctx, "HeapInuse", float64(a.memstats.HeapInuse))
	a.writer.SetGauge(ctx, "HeapObjects", float64(a.memstats.HeapObjects))
	a.writer.SetGauge(ctx, "HeapReleased", float64(a.memstats.HeapReleased))
	a.writer.SetGauge(ctx, "HeapSys", float64(a.memstats.HeapSys))
	a.writer.SetGauge(ctx, "LastGC", float64(a.memstats.LastGC))
	a.writer.SetGauge(ctx, "Lookups", float64(a.memstats.Lookups))
	a.writer.SetGauge(ctx, "MCacheInuse", float64(a.memstats.MCacheInuse))
	a.writer.SetGauge(ctx, "MCacheSys", float64(a.memstats.MCacheSys))
	a.writer.SetGauge(ctx, "Mallocs", float64(a.memstats.Mallocs))
	a.writer.SetGauge(ctx, "MSpanInuse", float64(a.memstats.MSpanInuse))
	a.writer.SetGauge(ctx, "MSpanSys", float64(a.memstats.MSpanSys))
	a.writer.SetGauge(ctx, "NextGC", float64(a.memstats.NextGC))
	a.writer.SetGauge(ctx, "NumForcedGC", float64(a.memstats.NumForcedGC))
	a.writer.SetGauge(ctx, "NumGC", float64(a.memstats.NumGC))
	a.writer.SetGauge(ctx, "OtherSys", float64(a.memstats.OtherSys))
	a.writer.SetGauge(ctx, "PauseTotalNs", float64(a.memstats.PauseTotalNs))
	a.writer.SetGauge(ctx, "StackInuse", float64(a.memstats.StackInuse))
	a.writer.SetGauge(ctx, "StackSys", float64(a.memstats.StackSys))
	a.writer.SetGauge(ctx, "Sys", float64(a.memstats.Sys))
	a.writer.SetGauge(ctx, "TotalAlloc", float64(a.memstats.TotalAlloc))
}

func (a *Agent) PollWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.PollInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			a.logger.Info("PollWorker: context canceled, stopping")
			return
		case <-ticker.C:
			a.GetMetrics(ctx)

		}
	}

}
func (a *Agent) ReportWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.ReportInterval) * time.Second)
	defer ticker.Stop()
	for {

		select {
		case <-ctx.Done():
			a.logger.Info("ReportWorker: context canceled, stopping")
			return
		case <-ticker.C:
			counter, err := a.reader.GetMapCounter(ctx)
			if err != nil {
				a.logger.Error("Ошибка получения счетчика:", zap.Error(err))
				continue
			}
			for nameCounter, valueCounter := range counter {
				err = a.PushCounter(nameCounter, valueCounter)
				if err != nil {
					a.logger.Error("Ошибка отправки счетчика:", zap.Error(err))
					continue
				}
			}
			gaugeMap, err := a.reader.GetMapGauge(ctx)
			if err != nil {
				a.logger.Error("Ошибка получения гауга:", zap.Error(err))
				continue
			}
			for nameGauge, valueGauge := range gaugeMap {
				err = a.PushGauge(nameGauge, valueGauge)
				if err != nil {
					a.logger.Error("Ошибка отправки гауга:", zap.Error(err))
					continue
				}
			}
		}
	}
}

func (a *Agent) ReportWorkerBatch(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.ReportInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("ReportWorkerBatch: context canceled, stopping")
			return
		case <-ticker.C:
			var metrics []models.Metrics

			// Добавляем счетчики
			counters, err := a.reader.GetMapCounter(ctx)
			if err != nil {
				a.logger.Error("Ошибка получения счетчиков:", zap.Error(err))
				continue
			}
			for name, value := range counters {
				delta := value
				metrics = append(metrics, models.Metrics{
					ID:    name,
					MType: "counter",
					Delta: &delta,
				})
			}

			// Добавляем gauge метрики
			gauges, err := a.reader.GetMapGauge(ctx)
			if err != nil {
				a.logger.Error("Ошибка получения gauge метрик:", zap.Error(err))
				continue
			}
			for name, value := range gauges {
				val := value
				metrics = append(metrics, models.Metrics{
					ID:    name,
					MType: "gauge",
					Value: &val,
				})
			}

			// Отправляем батч
			if len(metrics) > 0 {
				err := a.PushMetricsBatch(metrics)
				if err != nil {
					a.logger.Error("Ошибка отправки метрик:", zap.Error(err))
				}
			}

		}
	}
}

func (a *Agent) PushCounter(name string, value int64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}

	return a.sendCompressedMetric(&metric)
}

func (a *Agent) PushGauge(name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}

	return a.sendCompressedMetric(&metric)
}

// PushMetricsBatch отправляет метрики батчем на сервер
func (a *Agent) PushMetricsBatch(metrics []models.Metrics) error {
	// Проверяем, что батч не пустой
	if len(metrics) == 0 {
		return nil
	}

	// Конвертируем метрики в JSON
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}

	// Сжимаем данные
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// Создаем запрос
	url := fmt.Sprintf("http://%s/updates/", a.config.MetricServerHost)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	// Отправляем запрос
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push metrics batch: %s", resp.Status)
	}

	return nil
}

// sendCompressedMetric sends a metric in JSON format with gzip compression
func (a *Agent) sendCompressedMetric(metric *models.Metrics) error {
	// Convert metric to JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric to JSON: %w", err)
	}

	// Compress JSON data
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// Create request
	url := fmt.Sprintf("http://%s/update/", a.config.MetricServerHost)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	var lastErr error
	for _, backoff := range a.backoffSchedule {
		err := a.sendRequest(req)
		if err == nil {
			return nil // Success
		}
		lastErr = err
		a.logger.Error("failed to send request", zap.Error(err))
		a.logger.Info("retrying in", zap.Duration("dur", backoff))
		time.Sleep(backoff)

	}
	// All retries failed
	return fmt.Errorf("failed to send request after retries: %v", lastErr)
}

func (a *Agent) sendRequest(req *http.Request) error {
	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push metric: %s", resp.Status)
	}

	return nil
}

// compressData compresses data using gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
