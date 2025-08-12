package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/retry"
	"github.com/iudanet/yp-metrics-go/internal/utils"
)

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
	if a.config.SginKey != "" {
		hash := utils.CalculateHash(jsonData, a.config.SginKey)
		req.Header.Set("HashSHA256", hash)

	}
	// Отправляем запрос
	return retry.WithRetry(func() error {
		return a.sendRequest(req)
	})
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
	if a.config.SginKey != "" {
		hash := utils.CalculateHash(jsonData, a.config.SginKey)
		req.Header.Set("HashSHA256", hash)
	}
	return retry.WithRetry(func() error {
		return a.sendRequest(req)
	})
}

func (a *Agent) sendRequest(req *http.Request) error {
	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
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
