package server

import (
	"context"
	"errors"
	"testing"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var errTestStorage = errors.New("storage error")

func TestGRPCServer_UpdateMetric(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	store := localStore.New()
	server := NewGRPCServer(store, logger)

	tests := []struct {
		metric        *grpcmetrics.Metric
		name          string
		expectedError bool
	}{
		{
			name: "counter metric",
			metric: &grpcmetrics.Metric{
				Id:    "testCounter",
				Type:  grpcmetrics.MetricType_COUNTER,
				Delta: 42,
			},
			expectedError: false,
		},
		{
			name: "gauge metric",
			metric: &grpcmetrics.Metric{
				Id:    "testGauge",
				Type:  grpcmetrics.MetricType_GAUGE,
				Value: 3.14,
			},
			expectedError: false,
		},
		{
			name:          "empty metric",
			metric:        nil,
			expectedError: true,
		},
		{
			name: "invalid metric type",
			metric: &grpcmetrics.Metric{
				Id:   "testInvalid",
				Type: grpcmetrics.MetricType(999), // Invalid type
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &grpcmetrics.UpdateMetricRequest{Metric: tt.metric}
			resp, err := server.UpdateMetric(context.Background(), req)

			if tt.expectedError {
				assert.NoError(t, err) // gRPC server doesn't return errors, only success: false
				assert.False(t, resp.GetSuccess())
				assert.NotEmpty(t, resp.GetError())
			} else {
				assert.NoError(t, err)
				assert.True(t, resp.GetSuccess())
				assert.Empty(t, resp.GetError())

				// Verify the metric was stored
				ctx := context.Background()
				switch tt.metric.Type {
				case grpcmetrics.MetricType_COUNTER:
					value, err := store.GetCounter(ctx, tt.metric.Id)
					assert.NoError(t, err)
					assert.Equal(t, tt.metric.Delta, value)
				case grpcmetrics.MetricType_GAUGE:
					value, err := store.GetGauge(ctx, tt.metric.Id)
					assert.NoError(t, err)
					assert.Equal(t, tt.metric.Value, value)
				}
			}
		})
	}
}

func TestGRPCServer_UpdateMetricsBatch(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	store := localStore.New()
	server := NewGRPCServer(store, logger)

	tests := []struct {
		name          string
		metrics       []*grpcmetrics.Metric
		expectedError bool
	}{
		{
			name: "valid batch",
			metrics: []*grpcmetrics.Metric{
				{
					Id:    "counter1",
					Type:  grpcmetrics.MetricType_COUNTER,
					Delta: 100,
				},
				{
					Id:    "gauge1",
					Type:  grpcmetrics.MetricType_GAUGE,
					Value: 2.71,
				},
			},
			expectedError: false,
		},
		{
			name:          "empty batch",
			metrics:       []*grpcmetrics.Metric{},
			expectedError: true,
		},
		{
			name:          "nil batch",
			metrics:       nil,
			expectedError: true,
		},
		{
			name: "batch with invalid metric",
			metrics: []*grpcmetrics.Metric{
				{
					Id:    "counter1",
					Type:  grpcmetrics.MetricType_COUNTER,
					Delta: 100,
				},
				{
					Id:   "invalid",
					Type: grpcmetrics.MetricType(999), // Invalid type
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &grpcmetrics.UpdateMetricsBatchRequest{Metrics: tt.metrics}
			resp, err := server.UpdateMetricsBatch(context.Background(), req)

			if tt.expectedError {
				assert.NoError(t, err) // gRPC server doesn't return errors, only success: false
				assert.False(t, resp.GetSuccess())
				assert.NotEmpty(t, resp.GetError())
			} else {
				assert.NoError(t, err)
				assert.True(t, resp.GetSuccess())
				assert.Empty(t, resp.GetError())

				// Verify metrics were stored
				ctx := context.Background()
				for _, metric := range tt.metrics {
					switch metric.Type {
					case grpcmetrics.MetricType_COUNTER:
						value, err := store.GetCounter(ctx, metric.Id)
						assert.NoError(t, err)
						assert.Equal(t, metric.Delta, value)
					case grpcmetrics.MetricType_GAUGE:
						value, err := store.GetGauge(ctx, metric.Id)
						assert.NoError(t, err)
						assert.Equal(t, metric.Value, value)
					}
				}
			}
		})
	}
}

func TestGRPCServer_StorageErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a mock storage that returns errors
	mockStorage := &mockErrorStorage{}
	server := NewGRPCServer(mockStorage, logger)

	// Test counter error
	counterReq := &grpcmetrics.UpdateMetricRequest{
		Metric: &grpcmetrics.Metric{
			Id:    "testCounter",
			Type:  grpcmetrics.MetricType_COUNTER,
			Delta: 42,
		},
	}

	resp, err := server.UpdateMetric(context.Background(), counterReq)
	assert.NoError(t, err) // gRPC server should handle storage errors gracefully
	assert.False(t, resp.GetSuccess())
	assert.Contains(t, resp.GetError(), "storage error")

	// Test gauge error
	gaugeReq := &grpcmetrics.UpdateMetricRequest{
		Metric: &grpcmetrics.Metric{
			Id:    "testGauge",
			Type:  grpcmetrics.MetricType_GAUGE,
			Value: 3.14,
		},
	}

	resp, err = server.UpdateMetric(context.Background(), gaugeReq)
	assert.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.Contains(t, resp.GetError(), "storage error")
}

func TestGRPCServer_BatchStorageErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	mockStorage := &mockErrorStorage{}
	server := NewGRPCServer(mockStorage, logger)

	req := &grpcmetrics.UpdateMetricsBatchRequest{
		Metrics: []*grpcmetrics.Metric{
			{
				Id:    "counter1",
				Type:  grpcmetrics.MetricType_COUNTER,
				Delta: 100,
			},
		},
	}

	resp, err := server.UpdateMetricsBatch(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.Contains(t, resp.GetError(), "storage error")
}

// mockErrorStorage is a storage that always returns errors
type mockErrorStorage struct{}

func (m *mockErrorStorage) SetCounter(ctx context.Context, name string, value int64) error {
	return errTestStorage
}

func (m *mockErrorStorage) SetGauge(ctx context.Context, name string, value float64) error {
	return errTestStorage
}

func (m *mockErrorStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	return 0, errTestStorage
}

func (m *mockErrorStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	return 0, errTestStorage
}

func (m *mockErrorStorage) GetMapCounter(ctx context.Context) (map[string]int64, error) {
	return nil, errTestStorage
}

func (m *mockErrorStorage) GetMapGauge(ctx context.Context) (map[string]float64, error) {
	return nil, errTestStorage
}

func (m *mockErrorStorage) WriteBatch(ctx context.Context, metrics []models.Metrics) error {
	return errTestStorage
}

func (m *mockErrorStorage) SaveDB(ctx context.Context, path string) error {
	return errTestStorage
}

func (m *mockErrorStorage) LoadDB(ctx context.Context, path string) error {
	return errTestStorage
}

func (m *mockErrorStorage) Ping(ctx context.Context) error {
	return errTestStorage
}

func (m *mockErrorStorage) IncrCounter(ctx context.Context, name string) error {
	return errTestStorage
}
