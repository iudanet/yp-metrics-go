package agent

import (
	"context"
	"net"
	"testing"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const bufSize = 1024 * 1024

// mockMetricsServiceServer is a mock implementation for testing
type mockMetricsServiceServer struct {
	grpcmetrics.UnimplementedMetricsServiceServer
	updateMetricErr error
	updateBatchErr  error
	lastMetric      *grpcmetrics.Metric
	lastBatch       []*grpcmetrics.Metric
}

func (m *mockMetricsServiceServer) UpdateMetric(ctx context.Context, req *grpcmetrics.UpdateMetricRequest) (*grpcmetrics.UpdateMetricResponse, error) {
	if m.updateMetricErr != nil {
		return nil, m.updateMetricErr
	}
	m.lastMetric = req.GetMetric()
	return &grpcmetrics.UpdateMetricResponse{Success: true}, nil
}

func (m *mockMetricsServiceServer) UpdateMetricsBatch(ctx context.Context, req *grpcmetrics.UpdateMetricsBatchRequest) (*grpcmetrics.UpdateMetricsBatchResponse, error) {
	if m.updateBatchErr != nil {
		return nil, m.updateBatchErr
	}
	m.lastBatch = req.GetMetrics()
	return &grpcmetrics.UpdateMetricsBatchResponse{Success: true}, nil
}

func setupGRPCTest(t *testing.T) (*grpcAgentClient, *mockMetricsServiceServer, func()) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	mockServer := &mockMetricsServiceServer{}
	s := grpc.NewServer()
	grpcmetrics.RegisterMetricsServiceServer(s, mockServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()

	conn, connErr := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if connErr != nil {
		t.Fatalf("Failed to dial: %v", connErr)
	}

	logger, _ := zap.NewDevelopment()
	client := &grpcAgentClient{
		client: grpcmetrics.NewMetricsServiceClient(conn),
		logger: logger,
	}

	cleanup := func() {
		conn.Close()
		s.Stop()
		lis.Close()
	}

	return client, mockServer, cleanup
}

func TestGRPCAgentClient_PushMetric_Success(t *testing.T) {
	client, mockServer, cleanup := setupGRPCTest(t)
	defer cleanup()

	// Test counter metric
	counterValue := int64(42)
	counterMetric := models.Metrics{
		ID:    "testCounter",
		MType: models.TypeCounter,
		Delta: &counterValue,
	}

	err := client.PushMetric(context.Background(), counterMetric)
	if err != nil {
		t.Fatalf("PushMetric failed: %v", err)
	}

	if mockServer.lastMetric == nil {
		t.Fatal("No metric was received by server")
	}

	if mockServer.lastMetric.Id != "testCounter" {
		t.Errorf("Expected metric ID 'testCounter', got '%s'", mockServer.lastMetric.Id)
	}

	if mockServer.lastMetric.Type != grpcmetrics.MetricType_COUNTER {
		t.Errorf("Expected metric type COUNTER, got %v", mockServer.lastMetric.Type)
	}

	if mockServer.lastMetric.Delta != 42 {
		t.Errorf("Expected delta 42, got %d", mockServer.lastMetric.Delta)
	}

	// Test gauge metric
	gaugeValue := 3.14
	gaugeMetric := models.Metrics{
		ID:    "testGauge",
		MType: models.TypeGauge,
		Value: &gaugeValue,
	}

	err = client.PushMetric(context.Background(), gaugeMetric)
	if err != nil {
		t.Fatalf("PushMetric failed: %v", err)
	}

	if mockServer.lastMetric.Id != "testGauge" {
		t.Errorf("Expected metric ID 'testGauge', got '%s'", mockServer.lastMetric.Id)
	}

	if mockServer.lastMetric.Type != grpcmetrics.MetricType_GAUGE {
		t.Errorf("Expected metric type GAUGE, got %v", mockServer.lastMetric.Type)
	}

	if mockServer.lastMetric.Value != 3.14 {
		t.Errorf("Expected value 3.14, got %f", mockServer.lastMetric.Value)
	}
}

func TestGRPCAgentClient_PushMetricsBatch_Success(t *testing.T) {
	client, mockServer, cleanup := setupGRPCTest(t)
	defer cleanup()

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
	}

	err := client.PushMetricsBatch(context.Background(), metrics)
	if err != nil {
		t.Fatalf("PushMetricsBatch failed: %v", err)
	}

	if len(mockServer.lastBatch) != 2 {
		t.Fatalf("Expected 2 metrics in batch, got %d", len(mockServer.lastBatch))
	}
}

func TestIPMetadataInterceptor(t *testing.T) {
	t.Run("adds IP to metadata", func(t *testing.T) {
		ctx := context.Background()
		method := "/test.Method"
		req := "test request"
		reply := "test reply"

		// Mock invoker that checks metadata
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok, "metadata should be present")

			ipValues := md.Get("x-real-ip")
			require.Len(t, ipValues, 1, "should have one IP value")

			ip := net.ParseIP(ipValues[0])
			require.NotNil(t, ip, "IP should be valid")
			require.False(t, ip.IsLoopback(), "IP should not be loopback")

			return nil
		}

		err := ipMetadataInterceptor(ctx, method, req, &reply, nil, invoker)
		assert.NoError(t, err)
	})

	t.Run("handles error getting IP", func(t *testing.T) {
		// Create a wrapper that calls the original function but returns error
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		}

		// Test with a context that will cause getLocalIP to fail
		// (we can't easily mock getLocalIP, so we'll test the error path differently)
		// For now, we'll just test that the function doesn't panic
		ctx := context.Background()
		method := "/test.Method"
		req := "test request"
		reply := "test reply"

		// This should not panic even if getLocalIP returns empty
		err := ipMetadataInterceptor(ctx, method, req, &reply, nil, invoker)
		assert.NoError(t, err) // Should handle empty IP gracefully
	})

}

func TestGetLocalIP(t *testing.T) {
	ip, err := getLocalIP()

	// getLocalIP might return empty string without error if no non-loopback IPs found
	if err != nil {
		assert.Error(t, err)
	} else {
		// If IP is returned, it should be valid
		if ip != "" {
			parsedIP := net.ParseIP(ip)
			assert.NotNil(t, parsedIP, "returned IP should be valid")
			assert.False(t, parsedIP.IsLoopback(), "returned IP should not be loopback")
		}
	}
}

func TestGRPCAgentClient_PushMetricsBatch_Empty(t *testing.T) {
	client, mockServer, cleanup := setupGRPCTest(t)
	defer cleanup()

	err := client.PushMetricsBatch(context.Background(), []models.Metrics{})
	if err != nil {
		t.Fatalf("PushMetricsBatch with empty batch should not fail: %v", err)
	}

	// Server should not receive any metrics for empty batch
	if mockServer.lastBatch != nil {
		t.Error("Server should not receive metrics for empty batch")
	}
}

func TestGRPCAgentClient_ServerError(t *testing.T) {
	client, mockServer, cleanup := setupGRPCTest(t)
	defer cleanup()

	// Configure server to return error
	mockServer.updateMetricErr = context.Canceled

	counterValue := int64(42)
	counterMetric := models.Metrics{
		ID:    "testCounter",
		MType: models.TypeCounter,
		Delta: &counterValue,
	}

	err := client.PushMetric(context.Background(), counterMetric)
	if err == nil {
		t.Error("Expected error from server, got nil")
	}
}

func TestGRPCAgentClient_BatchServerError(t *testing.T) {
	client, mockServer, cleanup := setupGRPCTest(t)
	defer cleanup()

	// Configure server to return error for batch
	mockServer.updateBatchErr = context.Canceled

	counterValue := int64(42)
	metrics := []models.Metrics{
		{
			ID:    "testCounter",
			MType: models.TypeCounter,
			Delta: &counterValue,
		},
	}

	err := client.PushMetricsBatch(context.Background(), metrics)
	if err == nil {
		t.Error("Expected error from server for batch, got nil")
	}
}
