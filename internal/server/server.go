// Package server provides HTTP server implementation for the metrics service.
// It handles HTTP requests, metric storage operations, and includes middleware for compression, logging, and content validation.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strconv"
	"text/template"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"go.uber.org/zap"
)

func NewService(storage storage.Repository, cfg *config.ServerConfig, logger *zap.Logger, pg storage.Repository) *Service {
	return &Service{
		storage: storage,
		viewer:  storage,
		config:  cfg,
		logger:  logger,
		checker: pg,
	}
}

type Service struct {
	storage storage.MetricWriter
	viewer  storage.MetricReader
	config  *config.ServerConfig
	logger  *zap.Logger
	checker storage.HealthcheckDB
}
type IndexData struct {
	Counters map[string]int64
	Gauges   map[string]float64
}

func (s *Service) UpdateMetricJSON(w http.ResponseWriter, req *http.Request) {
	var metrics models.Metrics

	err := json.NewDecoder(req.Body).Decode(&metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch metrics.MType {
	case models.TypeCounter:
		if metrics.Delta == nil {
			http.Error(w, "delta is required for counter", http.StatusBadRequest)
			return
		}
		err := s.storage.SetCounter(req.Context(), metrics.ID, *metrics.Delta)
		if err != nil {
			s.logger.Error("Ошибка при установке counter", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			err := s.storage.SaveDB(req.Context(), s.config.Storage.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	case models.TypeGauge:
		if metrics.Value == nil {
			http.Error(w, "value is required for gauge", http.StatusBadRequest)
			return
		}
		err := s.storage.SetGauge(req.Context(), metrics.ID, *metrics.Value)
		if err != nil {
			s.logger.Error("Ошибка при установке gauge", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			err := s.storage.SaveDB(req.Context(), s.config.Storage.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (s *Service) UpdateMetric(w http.ResponseWriter, req *http.Request) {
	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")
	rawValue := req.PathValue("value")
	switch typeMetrics {
	case models.TypeGauge:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
		err = s.storage.SetGauge(req.Context(), name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			err := s.storage.SaveDB(req.Context(), s.config.Storage.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	case models.TypeCounter:
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
		err = s.storage.SetCounter(req.Context(), name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			err := s.storage.SaveDB(req.Context(), s.config.Storage.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (s *Service) GetMetricJSON(w http.ResponseWriter, req *http.Request) {
	var metrics models.Metrics

	err := json.NewDecoder(req.Body).Decode(&metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var resp models.Metrics
	switch metrics.MType {
	case models.TypeGauge:
		var value float64
		value, err = s.viewer.GetGauge(req.Context(), metrics.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		resp.ID = metrics.ID
		resp.MType = metrics.MType
		resp.Value = &value
	case models.TypeCounter:
		var delta int64
		delta, err = s.viewer.GetCounter(req.Context(), metrics.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		resp.ID = metrics.ID
		resp.MType = metrics.MType
		resp.Delta = &delta
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}

}

func (s *Service) GetMetric(w http.ResponseWriter, req *http.Request) {
	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")

	switch typeMetrics {
	case models.TypeGauge:
		value, err := s.viewer.GetGauge(req.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		fmt.Fprint(w, strconv.FormatFloat(value, 'f', -1, 64))
	case models.TypeCounter:
		value, err := s.viewer.GetCounter(req.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, "%d\n", value)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

func (s *Service) GetIndex(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
	counters, err := s.viewer.GetMapCounter(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	gauges, err := s.viewer.GetMapGauge(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := IndexData{
		Counters: counters,
		Gauges:   gauges,
	}

	tmpl := template.Must(template.New("index").Parse(indexTemplate))
	w.Header().Set("Content-Type", "text/html")
	// Рендерим шаблон
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

// функция для логики /ping проверющая подклчюение к базе данных.
func (s *Service) Ping(w http.ResponseWriter, r *http.Request) {
	err := s.checker.Ping(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) UpdateMetricsBatch(w http.ResponseWriter, req *http.Request) {
	var metrics []models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	if err := s.storage.WriteBatch(ctx, metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Синхронное сохранение если нужно
	if s.config.Storage.StoreInterval == 0 {
		err := s.storage.SaveDB(ctx, s.config.Storage.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// NewRouter creates and configures HTTP router with all middleware and handlers
func (s *Service) NewRouter() *http.ServeMux {
	m := http.NewServeMux()

	// Базовые обработчики
	m.Handle(`POST /update/{$}`, s.CheckContentType(
		s.DecryptionMiddleware(
			http.HandlerFunc(s.UpdateMetricJSON))))

	m.Handle(`POST /updates/{$}`, s.CheckContentType(
		s.DecryptionMiddleware(
			s.VerifyHash(
				http.HandlerFunc(s.UpdateMetricsBatch)))))
	m.Handle(`POST /value/{$}`, s.CheckContentType(http.HandlerFunc(s.GetMetricJSON)))

	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, s.UpdateMetric)
	m.HandleFunc(`GET /value/{typeMetrics}/{name}`, s.GetMetric)
	m.HandleFunc(`GET /ping`, s.Ping)
	m.HandleFunc(`GET /{$}`, s.GetIndex)

	// Add pprof routes
	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return m
}

// GetHandlerWithMiddleware returns the router with all middleware applied
func (s *Service) GetHandlerWithMiddleware() http.Handler {
	return s.VerifyIP(s.GzipMiddleware(s.WithLogging(s.NewRouter())))
}
