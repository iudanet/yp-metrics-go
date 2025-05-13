package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"go.uber.org/zap"
)

const (
	typeGauge   string = "gauge"
	typeCounter string = "counter"
)

func NewService(storage storage.Repository, cfg *config.ServerConfig, logger *zap.Logger) *service {
	return &service{
		storage: storage,
		viewer:  storage,
		config:  cfg,
		logger:  logger,
	}
}

type service struct {
	storage storage.MetricWriter
	viewer  storage.MetricReader
	config  *config.ServerConfig
	logger  *zap.Logger
}
type IndexData struct {
	Counters map[string]int64
	Gauges   map[string]float64
}

func (s *service) UpdateMetricJSON(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}
	var metrics models.Metrics

	err := json.NewDecoder(req.Body).Decode(&metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch metrics.MType {
	case typeCounter:
		err := s.storage.SetCounter(metrics.ID, *metrics.Delta)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			s.storage.SaveDB(s.config.Storage.Path)
		}
	case typeGauge:
		err = s.storage.SetGauge(metrics.ID, *metrics.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			s.storage.SaveDB(s.config.Storage.Path)
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (s *service) UpdateMetric(w http.ResponseWriter, req *http.Request) {
	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")
	rawValue := req.PathValue("value")
	log.Printf("Received metric: type=%s name=%s value=%s", typeMetrics, name, rawValue)
	switch typeMetrics {
	case typeGauge:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
		err = s.storage.SetGauge(name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			s.storage.SaveDB(s.config.Storage.Path)
		}
	case typeCounter:
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
		err = s.storage.SetCounter(name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.config.Storage.StoreInterval == 0 {
			s.storage.SaveDB(s.config.Storage.Path)
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (s *service) GetMetricJSON(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}
	var metrics models.Metrics

	err := json.NewDecoder(req.Body).Decode(&metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var resp models.Metrics
	switch metrics.MType {
	case typeGauge:
		value, err := s.viewer.GetGauge(metrics.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		resp.ID = metrics.ID
		resp.MType = metrics.MType
		resp.Value = &value
	case typeCounter:
		delta, err := s.viewer.GetCounter(metrics.ID)
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

func (s *service) GetMetric(w http.ResponseWriter, req *http.Request) {
	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")

	switch typeMetrics {
	case typeGauge:
		value, err := s.viewer.GetGauge(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		fmt.Fprint(w, strconv.FormatFloat(value, 'f', -1, 64))
	case typeCounter:
		value, err := s.viewer.GetCounter(name)
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

func (s *service) GetIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
	counters, err := s.viewer.GetMapCounter()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	gauges, err := s.viewer.GetMapGauge()
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
