package agent

import (
	"context"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcAgentClient struct {
	client grpcmetrics.MetricsServiceClient
	logger *zap.Logger
}

func NewGRPCAgentClient(grpcAddr string, logger *zap.Logger) (*grpcAgentClient, error) {
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := grpcmetrics.NewMetricsServiceClient(conn)
	return &grpcAgentClient{client: client, logger: logger}, nil
}

func (g *grpcAgentClient) PushMetric(ctx context.Context, metric models.Metrics) error {
	var grpcMetric grpcmetrics.Metric
	grpcMetric.Id = metric.ID

	switch metric.MType {
	case models.TypeCounter:
		grpcMetric.Type = grpcmetrics.MetricType_COUNTER
		if metric.Delta != nil {
			grpcMetric.Delta = *metric.Delta
		}
	case models.TypeGauge:
		grpcMetric.Type = grpcmetrics.MetricType_GAUGE
		if metric.Value != nil {
			grpcMetric.Value = *metric.Value
		}
	}

	_, err := g.client.UpdateMetric(ctx, &grpcmetrics.UpdateMetricRequest{Metric: &grpcMetric})
	if err != nil {
		g.logger.Error("grpc PushMetric failed", zap.Error(err))
		return err
	}
	return nil
}

func (g *grpcAgentClient) PushMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	var grpcMetrics []*grpcmetrics.Metric
	for _, metric := range metrics {
		grpcMetric := &grpcmetrics.Metric{
			Id: metric.ID,
		}
		switch metric.MType {
		case models.TypeCounter:
			grpcMetric.Type = grpcmetrics.MetricType_COUNTER
			if metric.Delta != nil {
				grpcMetric.Delta = *metric.Delta
			}
		case models.TypeGauge:
			grpcMetric.Type = grpcmetrics.MetricType_GAUGE
			if metric.Value != nil {
				grpcMetric.Value = *metric.Value
			}
		}
		grpcMetrics = append(grpcMetrics, grpcMetric)
	}

	_, err := g.client.UpdateMetricsBatch(ctx, &grpcmetrics.UpdateMetricsBatchRequest{Metrics: grpcMetrics})
	if err != nil {
		g.logger.Error("grpc PushMetricsBatch failed", zap.Error(err))
		return err
	}
	return nil
}
