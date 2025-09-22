package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/models"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMetricJSON(t *testing.T) {
	tests := []struct {
		metric      models.Metrics
		name        string
		expectError bool
		statusCode  int
	}{
		{
			name: "Valid_gauge",
			metric: models.Metrics{
				ID:    "testGauge",
				MType: "gauge",
				Value: ptrFloat64(10.5),
			},
			statusCode: http.StatusOK,
		},
		{
			name: "Valid_counter",
			metric: models.Metrics{
				ID:    "testCounter",
				MType: "counter",
				Delta: ptrInt64(5),
			},
			statusCode: http.StatusOK,
		},
		{
			name: "Invalid_type",
			metric: models.Metrics{
				ID:    "invalid",
				MType: "invalid",
			},
			expectError: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name: "Invalid_json",
			metric: models.Metrics{
				ID:    "test",
				MType: "gauge",
			},
			expectError: true,
			statusCode:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := localStore.New()
			cfg, err := config.ParseServerFlagsArgs([]string{})
			assert.NoError(t, err)
			logger, _ := logger.New("info")
			svc := NewService(store, cfg, logger, store)

			body, _ := json.Marshal(tt.metric)
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			svc.UpdateMetricJSON(w, req)

			if tt.name == "Invalid_json" {
				// Для теста с невалидным JSON создаем отдельный запрос
				req = httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader([]byte("invalid json")))
				req.Header.Set("Content-Type", "application/json")
				w = httptest.NewRecorder()
				svc.UpdateMetricJSON(w, req)
			}

			if tt.expectError {
				assert.Equal(t, tt.statusCode, w.Code)
			} else {
				assert.Equal(t, tt.statusCode, w.Code)

				// Проверяем запись в хранилище
				switch tt.metric.MType {
				case "gauge":
					value, _ := store.GetGauge(req.Context(), tt.metric.ID)
					assert.Equal(t, *tt.metric.Value, value)
				case "counter":
					value, _ := store.GetCounter(req.Context(), tt.metric.ID)
					assert.Equal(t, *tt.metric.Delta, value)
				}
			}
		})
	}
}

func TestUpdateMetricJSON_StorageErrors(t *testing.T) {
	tests := []struct {
		mockError  error
		name       string
		metricType string
		wantStatus int
	}{
		{
			name:       "storage_error_gauge",
			metricType: "gauge",
			mockError:  assert.AnError,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "storage_error_counter",
			metricType: "counter",
			mockError:  assert.AnError,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock storage с ошибкой
			store := &mockStorage{
				setGaugeFunc: func(ctx context.Context, name string, value float64) error {
					return tt.mockError
				},
				setCounterFunc: func(ctx context.Context, name string, value int64) error {
					return tt.mockError
				},
			}

			cfg := &config.ServerConfig{}
			logger, _ := logger.New("info")
			svc := NewService(store, cfg, logger, store)

			var metric models.Metrics
			if tt.metricType == "gauge" {
				metric = models.Metrics{
					ID:    "testGauge",
					MType: "gauge",
					Value: ptrFloat64(10.5),
				}
			} else {
				metric = models.Metrics{
					ID:    "testCounter",
					MType: "counter",
					Delta: ptrInt64(5),
				}
			}

			body, _ := json.Marshal(metric)
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			svc.UpdateMetricJSON(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestGetMetricJSON(t *testing.T) {
	tests := []struct {
		name        string
		metricType  string
		metricName  string
		expectError bool
	}{
		{
			name:       "Existing_gauge",
			metricType: "gauge",
			metricName: "testGauge",
		},
		{
			name:       "Existing_counter",
			metricType: "counter",
			metricName: "testCounter",
		},
		{
			name:        "Non-existent_metric",
			metricType:  "gauge",
			metricName:  "unknown",
			expectError: true,
		},
		{
			name:        "Invalid_type",
			metricType:  "invalid",
			metricName:  "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := localStore.New()
			cfg, err := config.ParseServerFlagsArgs([]string{})
			assert.NoError(t, err)
			logger, _ := logger.New("info")
			svc := NewService(store, cfg, logger, store)

			// Подготавливаем тестовые данные
			if tt.metricName == "testGauge" {
				store.SetGauge(context.Background(), tt.metricName, 10.5)
			} else if tt.metricName == "testCounter" {
				store.SetCounter(context.Background(), tt.metricName, 5)
			}

			reqMetric := models.Metrics{
				ID:    tt.metricName,
				MType: tt.metricType,
			}

			body, _ := json.Marshal(reqMetric)
			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			svc.GetMetricJSON(w, req)

			if tt.expectError {
				assert.NotEqual(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)

				var resp models.Metrics
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)

				assert.Equal(t, tt.metricName, resp.ID)
				assert.Equal(t, tt.metricType, resp.MType)

				if tt.metricType == "gauge" {
					assert.NotNil(t, resp.Value)
					assert.Equal(t, 10.5, *resp.Value)
				} else {
					assert.NotNil(t, resp.Delta)
					assert.Equal(t, int64(5), *resp.Delta)
				}
			}
		})
	}
}
