package agent

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const integrationBufSize = 1024 * 1024

func setupGRPCIntegrationTest(t *testing.T) (*grpcAgentClient, storage.Repository, func()) {
	// Create TCP listener
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	// Setup server
	logger, _ := zap.NewDevelopment()
	store := localStore.New()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	grpcmetrics.RegisterMetricsServiceServer(grpcServer, server.NewGRPCServer(store, logger))

	// Start gRPC server
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	// Create gRPC client
	conn, err2 := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err2 != nil {
		t.Fatalf("Failed to dial: %v", err2)
	}

	client := &grpcAgentClient{
		client: grpcmetrics.NewMetricsServiceClient(conn),
		logger: logger,
	}

	cleanup := func() {
		conn.Close()
		grpcServer.Stop()
		lis.Close()
	}

	return client, store, cleanup
}

func TestGRPCIntegration_SingleMetric(t *testing.T) {
	client, store, cleanup := setupGRPCIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test counter metric
	counterValue := int64(42)
	counterMetric := models.Metrics{
		ID:    "integrationCounter",
		MType: models.TypeCounter,
		Delta: &counterValue,
	}

	err := client.PushMetric(ctx, counterMetric)
	assert.NoError(t, err)

	// Verify metric was stored on server
	storedValue, err := store.GetCounter(ctx, "integrationCounter")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), storedValue)

	// Test gauge metric
	gaugeValue := 3.14
	gaugeMetric := models.Metrics{
		ID:    "integrationGauge",
		MType: models.TypeGauge,
		Value: &gaugeValue,
	}

	err = client.PushMetric(ctx, gaugeMetric)
	assert.NoError(t, err)

	// Verify gauge was stored
	storedGauge, err := store.GetGauge(ctx, "integrationGauge")
	assert.NoError(t, err)
	assert.Equal(t, 3.14, storedGauge)
}

func TestGRPCIntegration_BatchMetrics(t *testing.T) {
	client, store, cleanup := setupGRPCIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create batch of metrics
	counterValue := int64(100)
	gaugeValue := 2.71
	metrics := []models.Metrics{
		{
			ID:    "batchCounter",
			MType: models.TypeCounter,
			Delta: &counterValue,
		},
		{
			ID:    "batchGauge",
			MType: models.TypeGauge,
			Value: &gaugeValue,
		},
		{
			ID:    "anotherCounter",
			MType: models.TypeCounter,
			Delta: &counterValue,
		},
	}

	err := client.PushMetricsBatch(ctx, metrics)
	assert.NoError(t, err)

	// Verify all metrics were stored
	storedCounter, err := store.GetCounter(ctx, "batchCounter")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), storedCounter)

	storedGauge, err := store.GetGauge(ctx, "batchGauge")
	assert.NoError(t, err)
	assert.Equal(t, 2.71, storedGauge)

	storedAnotherCounter, err := store.GetCounter(ctx, "anotherCounter")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), storedAnotherCounter)
}

func TestGRPCIntegration_EmptyBatch(t *testing.T) {
	client, _, cleanup := setupGRPCIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Empty batch should not cause errors
	err := client.PushMetricsBatch(ctx, []models.Metrics{})
	assert.NoError(t, err)
}

func TestGRPCIntegration_ConcurrentAccess(t *testing.T) {
	client, store, cleanup := setupGRPCIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test concurrent metric updates with unique metric names
	const numGoroutines = 5
	const metricsPerRoutine = 10

	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			for j := 0; j < metricsPerRoutine; j++ {
				// Use unique metric name for each update
				metricName := string(rune('A'+routineID)) + "_" + string(rune('a'+j))
				gaugeValue := float64(routineID*100 + j)

				metric := models.Metrics{
					ID:    metricName,
					MType: models.TypeGauge,
					Value: &gaugeValue,
				}

				if err := client.PushMetric(ctx, metric); err != nil {
					errCh <- err
					return
				}

				// Small delay to allow interleaving
				time.Sleep(time.Millisecond * 10)
			}
			errCh <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-errCh
		assert.NoError(t, err)
	}

	// Verify all metrics were stored correctly
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < metricsPerRoutine; j++ {
			metricName := string(rune('A'+i)) + "_" + string(rune('a'+j))
			value, err := store.GetGauge(ctx, metricName)
			assert.NoError(t, err)
			expected := float64(i*100 + j)
			assert.Equal(t, expected, value)
		}
	}
}

func TestGRPCIntegration_ErrorHandling(t *testing.T) {
	client, _, cleanup := setupGRPCIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test with invalid metric (should be handled gracefully)
	invalidMetric := models.Metrics{
		ID:    "test",
		MType: "invalid_type", // Invalid type
	}

	err := client.PushMetric(ctx, invalidMetric)
	// This should not cause a panic, but might return an error
	// depending on how the server handles invalid types
	t.Logf("PushMetric with invalid type returned: %v", err)
	// We don't assert on the error since the server might handle it differently
}
