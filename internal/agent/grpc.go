package agent

import (
	"context"
	"net"

	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type grpcAgentClient struct {
	client grpcmetrics.MetricsServiceClient
	logger *zap.Logger
}

func NewGRPCAgentClient(grpcAddr string, logger *zap.Logger) (*grpcAgentClient, error) {
	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(ipMetadataInterceptor),
	)
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

// ipMetadataInterceptor adds client IP to gRPC metadata
func ipMetadataInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Get local IP address
	localIP, err := getLocalIP()
	if err != nil {
		return err
	}

	// Add IP to metadata
	md := metadata.Pairs("x-real-ip", localIP)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return invoker(ctx, method, req, reply, cc, opts...)
}

// getLocalIP returns the non-loopback local IP of the machine
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", nil
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
