package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
			tmpFile, err := os.CreateTemp("", "metrics-db-*.json")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			store := localStore.New()
			cfg := &config.ServerConfig{
				MetricServerHost: "localhost:8080",
				Storage: config.Storage{
					Restore:       false,
					Path:          tmpFile.Name(),
					StoreInterval: 0,
					DatabaseDSN:   "",
				},
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
	cfg, err := config.ParseServerFlagsArgs([]string{})
	assert.NoError(t, err)
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
			name: "valid_batch",
			metrics: []models.Metrics{
				{ID: "test1", MType: "gauge", Value: ptrFloat64(10.5)},
				{ID: "test2", MType: "counter", Delta: ptrInt64(5)},
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "empty_batch",
			metrics:    []models.Metrics{},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "invalid_metric_type",
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

			tmpFile, err := os.CreateTemp("", "metrics-db-*.json")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			store := localStore.New()
			cfg := &config.ServerConfig{
				MetricServerHost: "localhost:8080",
				Storage: config.Storage{
					Restore:       false,
					Path:          tmpFile.Name(),
					StoreInterval: 0,
					DatabaseDSN:   "",
				},
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

func TestGetMetric(t *testing.T) {
	tests := []struct {
		mockGaugeError   error
		mockCounterError error
		name             string
		urlPath          string
		wantBody         string
		mockGaugeValue   float64
		mockCounterValue int64
		wantStatus       int
	}{
		{
			name:           "valid_gauge_metric",
			urlPath:        "/value/gauge/testGauge",
			mockGaugeValue: 10.5,
			mockGaugeError: nil,
			wantStatus:     http.StatusOK,
			wantBody:       "10.5",
		},
		{
			name:             "valid_counter_metric",
			urlPath:          "/value/counter/testCounter",
			mockCounterValue: 42,
			mockCounterError: nil,
			wantStatus:       http.StatusOK,
			wantBody:         "42\n",
		},
		{
			name:           "gauge_not_found",
			urlPath:        "/value/gauge/nonexistent",
			mockGaugeValue: 0,
			mockGaugeError: errors.New("not found"),
			wantStatus:     http.StatusNotFound,
			wantBody:       "not found\n",
		},
		{
			name:             "counter_not_found",
			urlPath:          "/value/counter/nonexistent",
			mockCounterValue: 0,
			mockCounterError: errors.New("not found"),
			wantStatus:       http.StatusNotFound,
			wantBody:         "not found\n",
		},
		{
			name:       "invalid_metric_type",
			urlPath:    "/value/invalid/test",
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid metric type\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newLogger, err := logger.New("Info")
			assert.NoError(t, err)

			// Mock storage
			store := &mockStorage{
				getGaugeFunc: func(ctx context.Context, name string) (float64, error) {
					return tt.mockGaugeValue, tt.mockGaugeError
				},
				getCounterFunc: func(ctx context.Context, name string) (int64, error) {
					return tt.mockCounterValue, tt.mockCounterError
				},
			}

			cfg := &config.ServerConfig{}
			svc := NewService(store, cfg, newLogger, store)

			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)
			w := httptest.NewRecorder()

			// Создаем новый роутер и регистрируем обработчик
			mux := http.NewServeMux()
			mux.HandleFunc(`GET /value/{typeMetrics}/{name}`, svc.GetMetric)

			// Обрабатываем запрос
			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantBody, w.Body.String())
		})
	}
}

func TestGetIndex(t *testing.T) {
	tests := []struct {
		mockError    error
		mockCounters map[string]int64
		mockGauges   map[string]float64
		name         string
		urlPath      string
		wantContains []string
		wantStatus   int
	}{
		{
			name:         "valid_index",
			urlPath:      "/",
			mockCounters: map[string]int64{"counter1": 10, "counter2": 20},
			mockGauges:   map[string]float64{"gauge1": 1.5, "gauge2": 2.5},
			mockError:    nil,
			wantStatus:   http.StatusOK,
			wantContains: []string{"counter1", "counter2", "gauge1", "gauge2"},
		},
		{
			name:         "empty_metrics",
			urlPath:      "/",
			mockCounters: map[string]int64{},
			mockGauges:   map[string]float64{},
			mockError:    nil,
			wantStatus:   http.StatusOK,
			wantContains: []string{},
		},
		{
			name:         "storage_error",
			urlPath:      "/",
			mockCounters: nil,
			mockGauges:   nil,
			mockError:    errors.New("storage error"),
			wantStatus:   http.StatusNotFound,
			wantContains: []string{"storage error"},
		},
		{
			name:         "invalid_path",
			urlPath:      "/invalid",
			wantStatus:   http.StatusBadRequest,
			wantContains: []string{"invalid metric type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newLogger, err := logger.New("Info")
			assert.NoError(t, err)

			// Mock storage
			store := &mockStorage{
				getMapCounterFunc: func(ctx context.Context) (map[string]int64, error) {
					return tt.mockCounters, tt.mockError
				},
				getMapGaugeFunc: func(ctx context.Context) (map[string]float64, error) {
					return tt.mockGauges, tt.mockError
				},
			}

			cfg := &config.ServerConfig{}
			svc := NewService(store, cfg, newLogger, store)

			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)
			w := httptest.NewRecorder()

			svc.GetIndex(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			body := w.Body.String()
			for _, contain := range tt.wantContains {
				assert.Contains(t, body, contain)
			}
		})
	}
}

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name       string
		dbOnline   bool
		wantStatus int
	}{
		{
			name:       "db_online",
			dbOnline:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "db_offline",
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
	pingFunc          func(ctx context.Context) error
	getGaugeFunc      func(ctx context.Context, name string) (float64, error)
	getCounterFunc    func(ctx context.Context, name string) (int64, error)
	getMapCounterFunc func(ctx context.Context) (map[string]int64, error)
	getMapGaugeFunc   func(ctx context.Context) (map[string]float64, error)
	setGaugeFunc      func(ctx context.Context, name string, value float64) error
	setCounterFunc    func(ctx context.Context, name string, value int64) error
	incrCounterFunc   func(ctx context.Context, name string) error
	saveDBFunc        func(ctx context.Context, name string) error
	loadDBFunc        func(ctx context.Context, name string) error
	writeBatchFunc    func(ctx context.Context, metrics []models.Metrics) error
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return m.pingFunc(ctx)
}

func (m *mockStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	if m.getGaugeFunc != nil {
		return m.getGaugeFunc(ctx, name)
	}
	return 0, nil
}

func (m *mockStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	if m.getCounterFunc != nil {
		return m.getCounterFunc(ctx, name)
	}
	return 0, nil
}

func (m *mockStorage) GetMapCounter(ctx context.Context) (map[string]int64, error) {
	if m.getMapCounterFunc != nil {
		return m.getMapCounterFunc(ctx)
	}
	return nil, nil
}

func (m *mockStorage) GetMapGauge(ctx context.Context) (map[string]float64, error) {
	if m.getMapGaugeFunc != nil {
		return m.getMapGaugeFunc(ctx)
	}
	return nil, nil
}

func (m *mockStorage) SetGauge(ctx context.Context, name string, value float64) error {
	if m.setGaugeFunc != nil {
		return m.setGaugeFunc(ctx, name, value)
	}
	return nil
}

func (m *mockStorage) SetCounter(ctx context.Context, name string, value int64) error {
	if m.setCounterFunc != nil {
		return m.setCounterFunc(ctx, name, value)
	}
	return nil
}

func (m *mockStorage) IncrCounter(ctx context.Context, name string) error {
	if m.incrCounterFunc != nil {
		return m.incrCounterFunc(ctx, name)
	}
	return nil
}

func (m *mockStorage) SaveDB(ctx context.Context, name string) error {
	if m.saveDBFunc != nil {
		return m.saveDBFunc(ctx, name)
	}
	return nil
}

func (m *mockStorage) LoadDB(ctx context.Context, name string) error {
	if m.loadDBFunc != nil {
		return m.loadDBFunc(ctx, name)
	}
	return nil
}

func (m *mockStorage) WriteBatch(ctx context.Context, metrics []models.Metrics) error {
	if m.writeBatchFunc != nil {
		return m.writeBatchFunc(ctx, metrics)
	}
	return nil
}
