package server

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

func (s *Service) VerifyIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если подсеть не настроена, пропускаем проверку
		if s.config.TrustedSubnet == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Получаем IP из заголовка или из RemoteAddr
		ipStr := r.Header.Get("X-Real-IP")
		if ipStr == "" {
			// Fallback: извлекаем IP из RemoteAddr
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				s.logger.Warn("Failed to parse RemoteAddr", zap.Error(err))
				http.Error(w, "Invalid client address", http.StatusForbidden)
				return
			}
			ipStr = host
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			s.logger.Warn("Invalid IP address format", zap.String("ip", ipStr))
			http.Error(w, "Invalid IP address", http.StatusForbidden)
			return
		}

		s.logger.Debug("Received request", zap.String("ip", ip.String()))

		if !s.config.TrustedSubnet.Contains(ip) {
			s.logger.Warn("IP address not trusted",
				zap.String("ip", ip.String()),
				zap.String("trusted", s.config.TrustedSubnet.String()))
			http.Error(w, "IP address not trusted", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
