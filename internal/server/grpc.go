package server

import (
	"context"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"go.uber.org/zap"
)

// struct, реализующий grpcmetrics.MetricsServiceServer
type grpcServer struct {
	grpcmetrics.UnimplementedMetricsServiceServer
	storage storage.Repository
	logger  *zap.Logger
}

// Конструктор grpc сервера
func NewGRPCServer(storage storage.Repository, logger *zap.Logger) grpcmetrics.MetricsServiceServer {
	return &grpcServer{
		storage: storage,
		logger:  logger,
	}
}

// Метод UpdateMetric (отправка одной метрики)
func (s *grpcServer) UpdateMetric(ctx context.Context, req *grpcmetrics.UpdateMetricRequest) (*grpcmetrics.UpdateMetricResponse, error) {
	metric := req.GetMetric()
	if metric == nil {
		return &grpcmetrics.UpdateMetricResponse{
			Success: false,
			Error:   "empty metric",
		}, nil
	}

	switch metric.Type {
	case grpcmetrics.MetricType_COUNTER:
		err := s.storage.SetCounter(ctx, metric.Id, metric.Delta)
		if err != nil {
			s.logger.Error("failed to set counter", zap.Error(err))
			return &grpcmetrics.UpdateMetricResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
	case grpcmetrics.MetricType_GAUGE:
		err := s.storage.SetGauge(ctx, metric.Id, metric.Value)
		if err != nil {
			s.logger.Error("failed to set gauge", zap.Error(err))
			return &grpcmetrics.UpdateMetricResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
	default:
		return &grpcmetrics.UpdateMetricResponse{
			Success: false,
			Error:   "invalid metric type",
		}, nil
	}
	return &grpcmetrics.UpdateMetricResponse{Success: true}, nil
}

// Метод UpdateMetricsBatch (отправка батча метрик)
func (s *grpcServer) UpdateMetricsBatch(ctx context.Context, req *grpcmetrics.UpdateMetricsBatchRequest) (*grpcmetrics.UpdateMetricsBatchResponse, error) {
	metrics := req.GetMetrics()
	if len(metrics) == 0 {
		return &grpcmetrics.UpdateMetricsBatchResponse{
			Success: false,
			Error:   "empty metric batch",
		}, nil
	}

	var modelMetrics []models.Metrics
	for _, m := range metrics {
		var modelMetric models.Metrics
		switch m.Type {
		case grpcmetrics.MetricType_COUNTER:
			modelMetric = models.Metrics{
				ID:    m.Id,
				MType: models.TypeCounter,
				Delta: &m.Delta,
			}
		case grpcmetrics.MetricType_GAUGE:
			modelMetric = models.Metrics{
				ID:    m.Id,
				MType: models.TypeGauge,
				Value: &m.Value,
			}
		default:
			return &grpcmetrics.UpdateMetricsBatchResponse{
				Success: false,
				Error:   "invalid metric type in batch",
			}, nil
		}
		modelMetrics = append(modelMetrics, modelMetric)
	}

	err := s.storage.WriteBatch(ctx, modelMetrics)
	if err != nil {
		s.logger.Error("failed to write batch metrics", zap.Error(err))
		return &grpcmetrics.UpdateMetricsBatchResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &grpcmetrics.UpdateMetricsBatchResponse{
		Success: true,
	}, nil
}
