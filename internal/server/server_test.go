package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMetric(t *testing.T) {
	tests := []struct {
		name        string
		urlPath     string
		contentType string
		wantStatus  int
	}{
		{
			name:        "valid_gauge_metric",
			urlPath:     "/update/gauge/test/10.5",
			contentType: "text/plain",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "valid_counter_metric",
			urlPath:     "/update/counter/test/10",
			contentType: "text/plain",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "invalid_metric_type",
			urlPath:     "/update/invalid/test/10",
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid_gauge_value",
			urlPath:     "/update/gauge/test/invalid",
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid_content_type",
			urlPath:     "/update/gauge/test/10.5",
			contentType: "application/json",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "empty_metric_name",
			urlPath:     "/update/gauge//10.5",
			contentType: "text/plain",
			wantStatus:  http.StatusMovedPermanently,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newLogger, err := logger.New("Info")
			assert.NoError(t, err)

			store := localStore.New()
			cfg := &config.ServerConfig{
				MetricServerHost: "localhost:8080",
			}
			svc := NewService(store, cfg, newLogger, store)

			req := httptest.NewRequest(http.MethodPost, tt.urlPath, nil)
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			// Создаем новый роутер и регистрируем обработчик
			mux := http.NewServeMux()
			mux.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)

			// Обрабатываем запрос
			mux.ServeHTTP(w, req)

			// Проверяем результат
			if w.Code != tt.wantStatus {
				t.Errorf("UpdateMetric() status = %v, want %v", w.Code, tt.wantStatus)
				t.Logf("Response body: %v", w.Body.String())
			}
		})
	}
}

// Вспомогательная функция для проверки успешного обновления метрики
func TestUpdateMetricSuccess(t *testing.T) {
	store := localStore.New()
	cfg := config.NewServerConfig()
	newLogger, err := logger.New("Info")
	assert.NoError(t, err)
	svc := NewService(store, cfg, newLogger, store)

	tests := []struct {
		name       string
		metricType string
		metricName string
		value      string
	}{
		{
			name:       "gauge_update",
			metricType: "gauge",
			metricName: "testGauge",
			value:      "123.45",
		},
		{
			name:       "counter_update",
			metricType: "counter",
			metricName: "testCounter",
			value:      "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/update/%s/%s/%s",
				tt.metricType, tt.metricName, tt.value)

			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			mux := http.NewServeMux()
			mux.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)

			mux.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code,
				"Expected status code %d, got %d", http.StatusOK, w.Code)
		})
	}
}

func TestUpdateMetricsBatch(t *testing.T) {
	tests := []struct {
		name       string
		metrics    []models.Metrics
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid batch",
			metrics: []models.Metrics{
				{ID: "test1", MType: "gauge", Value: ptrFloat64(10.5)},
				{ID: "test2", MType: "counter", Delta: ptrInt64(5)},
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "empty batch",
			metrics:    []models.Metrics{},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "invalid metric type",
			metrics: []models.Metrics{
				{ID: "test1", MType: "invalid", Value: ptrFloat64(10.5)},
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newLogger, err := logger.New("Info")
			assert.NoError(t, err)

			store := localStore.New()
			cfg := &config.ServerConfig{
				MetricServerHost: "localhost:8080",
			}
			svc := NewService(store, cfg, newLogger, store)

			// Prepare request
			body, err := json.Marshal(tt.metrics)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			svc.UpdateMetricsBatch(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if !tt.wantError {
				var response []models.Metrics
				err = json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, len(tt.metrics), len(response))
			}
		})
	}
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name       string
		dbOnline   bool
		wantStatus int
	}{
		{
			name:       "db online",
			dbOnline:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "db offline",
			dbOnline:   false,
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newLogger, err := logger.New("Info")
			assert.NoError(t, err)

			// Mock storage
			store := &mockStorage{
				pingFunc: func(ctx context.Context) error {
					if tt.dbOnline {
						return nil
					}
					return errors.New("db offline")
				},
			}

			cfg := &config.ServerConfig{}
			svc := NewService(store, cfg, newLogger, store)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			svc.Ping(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

type mockStorage struct {
	storage.Repository
	pingFunc func(ctx context.Context) error
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return m.pingFunc(ctx)
}
