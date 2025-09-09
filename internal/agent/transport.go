package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iudanet/yp-metrics-go/internal/encryption"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/retry"
	"github.com/iudanet/yp-metrics-go/internal/utils"
)

// PushCounter отправляет значение counter на сервер
func (a *Agent) PushCounter(name string, value int64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}

	return a.sendSingleMetric(&metric)
}

// PushGauge отправляет значение gauge на сервер
func (a *Agent) PushGauge(name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}

	return a.sendSingleMetric(&metric)
}

// sendSingleMetric отправляет одиночную метрику на сервер
func (a *Agent) sendSingleMetric(metric *models.Metrics) error {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric to JSON: %w", err)
	}

	endpoint := "/update/"

	return a.sendData(jsonData, endpoint)
}

// PushMetricsBatch отправляет метрики батчем на сервер
func (a *Agent) PushMetricsBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}

	endpoint := "/updates/"

	return a.sendData(jsonData, endpoint)
}

// sendData отправляет данные на указанный endpoint с учетом шифрования
func (a *Agent) sendData(jsonData []byte, endpoint string) error {
	var dataToSend []byte
	var err error

	// Если настроено шифрование, шифруем данные
	if a.config.RSAPublicKeyPath != "" {
		dataToSend, err = encryption.Hybrid(jsonData, a.config.RSAPublicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
	} else {
		// Иначе используем обычные данные
		dataToSend = jsonData
	}

	// Сжимаем данные
	compressedData, err := compressData(dataToSend)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	url := fmt.Sprintf("http://%s%s", a.config.MetricServerHost, endpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	a.setRequestHeaders(req, jsonData)

	return retry.WithRetry(func() error {
		return a.sendRequest(req)
	})
}

// setRequestHeaders устанавливает заголовки для запроса
func (a *Agent) setRequestHeaders(req *http.Request, jsonData []byte) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if a.config.SginKey != "" {
		hash := utils.CalculateHash(jsonData, a.config.SginKey)
		req.Header.Set("HashSHA256", hash)
	}
}

// sendRequest выполняет HTTP запрос
func (a *Agent) sendRequest(req *http.Request) error {
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
