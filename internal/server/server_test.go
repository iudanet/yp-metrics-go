package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMetric(t *testing.T) {
	tests := []struct {
		name        string
		urlPath     string
		contentType string
		wantStatus  int
	}{
		{
			name:        "valid gauge metric",
			urlPath:     "/update/gauge/test/10.5",
			contentType: "text/plain",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "valid counter metric",
			urlPath:     "/update/counter/test/10",
			contentType: "text/plain",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "invalid metric type",
			urlPath:     "/update/invalid/test/10",
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid gauge value",
			urlPath:     "/update/gauge/test/invalid",
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid content type",
			urlPath:     "/update/gauge/test/10.5",
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "empty metric name",
			urlPath:     "/update/gauge//10.5",
			contentType: "text/plain",
			wantStatus:  http.StatusMovedPermanently,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storage.NewStorage()
			svc := NewService(store)

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
	store := storage.NewStorage()
	svc := NewService(store)

	tests := []struct {
		name       string
		metricType string
		metricName string
		value      string
	}{
		{
			name:       "gauge update",
			metricType: "gauge",
			metricName: "testGauge",
			value:      "123.45",
		},
		{
			name:       "counter update",
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
