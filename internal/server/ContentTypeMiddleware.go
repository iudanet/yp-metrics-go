package server

import (
	"net/http"
	"strings"
)

// Middleware для проверки Content-Type
func (s *Service) CheckContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
			http.Error(w, "invalid content type", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}
