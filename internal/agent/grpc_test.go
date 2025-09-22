package agent

import (
	"context"
	"net"
	"testing"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// Check counter metric
	counterMetric := mockServer.lastBatch[0]
	if counterMetric.Id != "batchCounter" {
		t.Errorf("Expected metric ID 'batchCounter', got '%s'", counterMetric.Id)
	}
	if counterMetric.Type != grpcmetrics.MetricType_COUNTER {
		t.Errorf("Expected metric type COUNTER, got %v", counterMetric.Type)
	}
	if counterMetric.Delta != 100 {
		t.Errorf("Expected delta 100, got %d", counterMetric.Delta)
	}

	// Check gauge metric
	gaugeMetric := mockServer.lastBatch[1]
	if gaugeMetric.Id != "batchGauge" {
		t.Errorf("Expected metric ID 'batchGauge', got '%s'", gaugeMetric.Id)
	}
	if gaugeMetric.Type != grpcmetrics.MetricType_GAUGE {
		t.Errorf("Expected metric type GAUGE, got %v", gaugeMetric.Type)
	}
	if gaugeMetric.Value != 2.71 {
		t.Errorf("Expected value 2.71, got %f", gaugeMetric.Value)
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
