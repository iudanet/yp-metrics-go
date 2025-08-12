package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/server"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func ExampleService_UpdateMetricJSON() {
	// Инициализируем зависимости
	store := localStore.New()
	cfg := &config.ServerConfig{
		Storage: config.Storage{
			StoreInterval: 10, // Не сохраняем сразу на диск
		},
	}
	logger := zap.NewNop() // Логгер без вывода

	// Создаем сервис с зависимостями
	svc := server.NewService(store, cfg, logger, store)

	// Подготавливаем метрику для отправки
	metric := models.Metrics{
		ID:    "testMetric",
		MType: "gauge",
		Value: ptrFloat64(10.5),
	}

	// Кодируем метрику в JSON
	body, _ := json.Marshal(metric)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Вызываем обработчик
	svc.UpdateMetricJSON(w, req)

	// Проверяем результат
	value, _ := store.GetGauge(context.Background(), "testMetric")
	fmt.Println("Status Code:", w.Code)
	fmt.Printf("Metric Value: %.1f\n", value)

	// Output:
	// Status Code: 200
	// Metric Value: 10.5
}

func ExampleService_GetMetricJSON() {
	// Инициализируем зависимости
	store := localStore.New()
	cfg := &config.ServerConfig{}
	logger := zap.NewNop()

	// Создаем сервис с зависимостями
	svc := server.NewService(store, cfg, logger, store)

	// Устанавливаем тестовые данные
	store.SetGauge(context.Background(), "existingMetric", 15.3)

	// Подготавливаем запрос метрики
	metric := models.Metrics{
		ID:    "existingMetric",
		MType: "gauge",
	}

	// Кодируем запрос в JSON
	body, _ := json.Marshal(metric)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Вызываем обработчик
	svc.GetMetricJSON(w, req)

	// Декодируем ответ
	var resp models.Metrics
	json.NewDecoder(w.Body).Decode(&resp)

	fmt.Println("Status Code:", w.Code)
	fmt.Printf("Metric: %s (type: %s) = %.1f\n", resp.ID, resp.MType, *resp.Value)

	// Output:
	// Status Code: 200
	// Metric: existingMetric (type: gauge) = 15.3
}

func ExampleService_UpdateMetricsBatch() {
	// Инициализируем зависимости
	store := localStore.New()
	cfg := &config.ServerConfig{
		Storage: config.Storage{
			StoreInterval: 10, // Не сохраняем сразу на диск
		},
	}
	logger := zap.NewNop()

	// Создаем сервис с зависимостями
	svc := server.NewService(store, cfg, logger, store)

	// Подготавливаем пакет метрик
	metrics := []models.Metrics{
		{
			ID:    "batchGauge",
			MType: "gauge",
			Value: ptrFloat64(12.5),
		},
		{
			ID:    "batchCounter",
			MType: "counter",
			Delta: ptrInt64(3),
		},
	}

	// Кодируем метрики в JSON
	body, _ := json.Marshal(metrics)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Вызываем обработчик
	svc.UpdateMetricsBatch(w, req)

	// Проверяем результат
	gauge, _ := store.GetGauge(context.Background(), "batchGauge")
	counter, _ := store.GetCounter(context.Background(), "batchCounter")

	fmt.Println("Status Code:", w.Code)
	fmt.Printf("Gauge: %.1f, Counter: %d\n", gauge, counter)

	// Output:
	// Status Code: 200
	// Gauge: 12.5, Counter: 3
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }

// TestExamples запускает все примеры как тесты
func TestExamples(t *testing.T) {
	// Инициализируем зависимости один раз
	store := localStore.New()
	cfg := &config.ServerConfig{
		Storage: config.Storage{
			StoreInterval: 10,
		},
	}
	logger := zap.NewNop()
	svc := server.NewService(store, cfg, logger, store)

	t.Run("UpdateMetricJSON", func(t *testing.T) {
		metric := models.Metrics{
			ID:    "testExample",
			MType: "gauge",
			Value: ptrFloat64(7.5),
		}

		body, _ := json.Marshal(metric)
		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		svc.UpdateMetricJSON(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		value, err := store.GetGauge(context.Background(), "testExample")
		assert.NoError(t, err)
		assert.Equal(t, 7.5, value)
	})

	t.Run("GetMetricJSON", func(t *testing.T) {
		store.SetGauge(context.Background(), "testGet", 9.9)

		metric := models.Metrics{
			ID:    "testGet",
			MType: "gauge",
		}

		body, _ := json.Marshal(metric)
		req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		svc.GetMetricJSON(w, req)

		var resp models.Metrics
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "testGet", resp.ID)
		assert.Equal(t, "gauge", resp.MType)
		assert.Equal(t, 9.9, *resp.Value)
	})

	t.Run("UpdateMetricsBatch", func(t *testing.T) {
		metrics := []models.Metrics{
			{
				ID:    "batchTestGauge",
				MType: "gauge",
				Value: ptrFloat64(11.1),
			},
			{
				ID:    "batchTestCounter",
				MType: "counter",
				Delta: ptrInt64(5),
			},
		}

		body, _ := json.Marshal(metrics)
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		svc.UpdateMetricsBatch(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		gauge, _ := store.GetGauge(context.Background(), "batchTestGauge")
		counter, _ := store.GetCounter(context.Background(), "batchTestCounter")
		assert.Equal(t, 11.1, gauge)
		assert.Equal(t, int64(5), counter)
	})
}
