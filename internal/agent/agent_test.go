package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
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

// Глобальная переменная для хранения последней полученной метрики в тестах
var receivedMetric *models.Metrics

func TestAgent(t *testing.T) {
	// Создаем тестовый HTTP сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем заголовки
		contentType := r.Header.Get("Content-Type")
		contentEncoding := r.Header.Get("Content-Encoding")
		
		// Проверяем, что запрос содержит JSON и сжат
		if contentType == "application/json" && contentEncoding == "gzip" {
			// Распаковываем gzip
			var reader io.Reader
			reader = r.Body
			if contentEncoding == "gzip" {
				gzipReader, err := gzip.NewReader(reader)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				defer gzipReader.Close()
				reader = gzipReader
			}
			
			// Декодируем JSON
			var metric models.Metrics
			if err := json.NewDecoder(reader).Decode(&metric); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			
			// Проверяем, что метрика содержит необходимые данные
			if metric.ID == "" || metric.MType == "" {
				http.Error(w, "invalid metric data", http.StatusBadRequest)
				return
			}
			
			// Сохраняем полученную метрику для проверки в тестах
			receivedMetric = &metric
		} else if !strings.HasPrefix(r.URL.Path, "/update/") {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name string
		fn   func(t *testing.T, a *Agent)
	}{
		{
			name: "test_metrics_collection",
			fn: func(t *testing.T, a *Agent) {
				a.GetMetrics()

				gauges, err := a.reader.GetMapGauge()
				require.NoError(t, err)
				assert.NotEmpty(t, gauges)

				counters, err := a.reader.GetMapCounter()
				require.NoError(t, err)
				assert.Equal(t, int64(1), counters["PollCount"])
			},
		},
		{
			name: "test_push_counter",
			fn: func(t *testing.T, a *Agent) {
				// Сбрасываем полученную метрику перед тестом
				receivedMetric = nil
				
				err := a.PushCounter("test", 10)
				assert.NoError(t, err)
				
				// Проверяем, что метрика была получена сервером
				assert.NotNil(t, receivedMetric, "Server should receive the metric")
				assert.Equal(t, "test", receivedMetric.ID)
				assert.Equal(t, "counter", receivedMetric.MType)
				assert.NotNil(t, receivedMetric.Delta)
				assert.Equal(t, int64(10), *receivedMetric.Delta)
			},
		},
		{
			name: "test_push_gauge",
			fn: func(t *testing.T, a *Agent) {
				// Сбрасываем полученную метрику перед тестом
				receivedMetric = nil
				
				err := a.PushGauge("test", 10.5)
				assert.NoError(t, err)
				
				// Проверяем, что метрика была получена сервером
				assert.NotNil(t, receivedMetric, "Server should receive the metric")
				assert.Equal(t, "test", receivedMetric.ID)
				assert.Equal(t, "gauge", receivedMetric.MType)
				assert.NotNil(t, receivedMetric.Value)
				assert.Equal(t, 10.5, *receivedMetric.Value)
			},
		},
		{
			name: "test_compressed_json_format",
			fn: func(t *testing.T, a *Agent) {
				// Сбрасываем полученную метрику перед тестом
				receivedMetric = nil
				
				// Отправляем метрику
				testValue := float64(42.42)
				err := a.PushGauge("compressed_json_test", testValue)
				assert.NoError(t, err)
				
				// Проверяем содержимое отправленной метрики
				assert.NotNil(t, receivedMetric, "Server should receive the compressed metric")
				assert.Equal(t, "compressed_json_test", receivedMetric.ID)
				assert.Equal(t, "gauge", receivedMetric.MType)
				assert.NotNil(t, receivedMetric.Value)
				assert.Equal(t, testValue, *receivedMetric.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AgentConfig{
				PollInterval:     2,
				ReportInterval:   10,
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
