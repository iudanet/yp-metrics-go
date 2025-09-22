package server

import (
	"context"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor returns a gRPC unary server interceptor for logging
func (s *Service) LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Log request details
		s.logger.Debug("gRPC request started",
			zap.String("method", info.FullMethod),
		)

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}

		// Log the request with detailed information
		logFields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
		}

		if err != nil {
			logFields = append(logFields, zap.Error(err))
			s.logger.Warn("gRPC request completed with error", logFields...)
		} else {
			s.logger.Info("gRPC request completed successfully", logFields...)
		}

		return resp, err
	}
}

// WithLoggingInterceptor returns a grpc.ServerOption with logging interceptor
func (s *Service) WithLoggingInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(s.LoggingInterceptor())
}

// IPVerificationInterceptor returns a gRPC unary server interceptor for IP verification
func (s *Service) IPVerificationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если подсеть не настроена, пропускаем проверку
		if s.config.TrustedSubnet == nil {
			return handler(ctx, req)
		}

		// Получаем IP из метаданных или из peer информации
		var ipStr string

		// Проверяем метаданные для X-Real-IP
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if realIP := md.Get("x-real-ip"); len(realIP) > 0 {
				ipStr = realIP[0]
			}
		}

		// Если не нашли в метаданных, проверяем peer информацию
		if ipStr == "" {
			if p, ok := peer.FromContext(ctx); ok {
				ipStr = p.Addr.String()
				// Извлекаем только IP адрес (убираем порт)
				if host, _, err := net.SplitHostPort(ipStr); err == nil {
					ipStr = host
				}
			}
		}

		// Если IP не найден, логируем предупреждение и пропускаем проверку
		// (вместо возврата ошибки, чтобы не ломать существующие клиенты)
		if ipStr == "" {
			s.logger.Warn("Client IP not found in metadata or peer info, skipping IP verification")
			return handler(ctx, req)
		}

		// Парсим IP адрес
		ip := net.ParseIP(ipStr)
		if ip == nil {
			s.logger.Warn("Invalid client IP format", zap.String("ip", ipStr))
			return handler(ctx, req)
		}

		// Проверяем, находится ли IP в доверенной подсети
		if !s.config.TrustedSubnet.Contains(ip) {
			s.logger.Warn("Access denied for IP", zap.String("ip", ipStr), zap.String("trusted_subnet", s.config.TrustedSubnet.String()))
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}

		s.logger.Debug("IP verification passed", zap.String("ip", ipStr))
		return handler(ctx, req)
	}
}

// WithIPVerificationInterceptor returns a grpc.ServerOption with IP verification interceptor
func (s *Service) WithIPVerificationInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(s.IPVerificationInterceptor())
}

// WithAllInterceptors returns a slice of grpc.ServerOption with all interceptors
func (s *Service) WithAllInterceptors() []grpc.ServerOption {
	return []grpc.ServerOption{
		s.WithLoggingInterceptor(),
		s.WithIPVerificationInterceptor(),
	}
}
