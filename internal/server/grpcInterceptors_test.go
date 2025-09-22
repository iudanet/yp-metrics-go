package server

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iudanet/yp-metrics-go/internal/config"
	mockStoragePkg "github.com/iudanet/yp-metrics-go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// mockHandler is a mock gRPC handler for testing
type mockHandler struct {
	response interface{}
	err      error
}

func (m *mockHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	return m.response, m.err
}

// TestService is a test wrapper for Service
type TestService struct {
	*Service
	logger *zap.Logger
}

func newTestService(cfg *config.ServerConfig, ctrl *gomock.Controller) *TestService {
	logger := zaptest.NewLogger(&testing.T{})
	mockStorage := mockStoragePkg.NewMockRepository(ctrl)

	return &TestService{
		Service: NewService(mockStorage, cfg, logger, mockStorage),
		logger:  logger,
	}
}

func TestLoggingInterceptor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.ServerConfig{}
	svc := newTestService(cfg, ctrl)
	interceptor := svc.LoggingInterceptor()

	// Test successful request
	mockHandler := &mockHandler{
		response: "test response",
		err:      nil,
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	ctx := context.Background()

	resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

	assert.NoError(t, err)
	assert.Equal(t, "test response", resp)

	// Test request with error
	expectedErr := status.Error(codes.Internal, "internal error")
	mockHandler.err = expectedErr
	mockHandler.response = nil

	resp, err = interceptor(ctx, "test request", info, mockHandler.handle)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, resp)
}

func TestIPVerificationInterceptor_NoTrustedSubnet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.ServerConfig{} // No trusted subnet configured
	svc := newTestService(cfg, ctrl)
	interceptor := svc.IPVerificationInterceptor()

	mockHandler := &mockHandler{response: "success"}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	ctx := context.Background()

	// Should pass through without verification
	resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestIPVerificationInterceptor_WithTrustedSubnet(t *testing.T) {
	// Create trusted subnet 192.168.1.0/24
	_, trustedNet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	ipnets := config.IPNets{*trustedNet}
	cfg := &config.ServerConfig{TrustedSubnet: ipnets}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := newTestService(cfg, ctrl)
	interceptor := svc.IPVerificationInterceptor()

	mockHandler := &mockHandler{response: "success"}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}

	t.Run("IP in trusted subnet via metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100"))

		resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("IP not in trusted subnet via metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "10.0.0.1"))

		resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

		assert.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Nil(t, resp)
	})

	t.Run("IP in trusted subnet via peer", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.50"), Port: 12345},
		})

		resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("Invalid IP format", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "invalid-ip"))

		resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

		assert.NoError(t, err) // Should pass through with warning
		assert.Equal(t, "success", resp)
	})

	t.Run("No IP found", func(t *testing.T) {
		ctx := context.Background() // No metadata or peer info

		resp, err := interceptor(ctx, "test request", info, mockHandler.handle)

		assert.NoError(t, err) // Should pass through with warning
		assert.Equal(t, "success", resp)
	})
}

func TestWithLoggingInterceptor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.ServerConfig{}
	svc := newTestService(cfg, ctrl)

	serverOption := svc.WithLoggingInterceptor()
	assert.NotNil(t, serverOption)
}

func TestWithIPVerificationInterceptor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.ServerConfig{}
	svc := newTestService(cfg, ctrl)

	serverOption := svc.WithIPVerificationInterceptor()
	assert.NotNil(t, serverOption)
}

func TestWithAllInterceptors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.ServerConfig{}
	svc := newTestService(cfg, ctrl)

	serverOptions := svc.WithAllInterceptors()
	assert.Len(t, serverOptions, 2)
	assert.NotNil(t, serverOptions[0])
	assert.NotNil(t, serverOptions[1])
}
