package service

import (
	"net/http"
	"strconv"

	"github.com/iudanet/yp-metrics-go/internal/repo"
)

type Service interface {
	UpdateMetric(w http.ResponseWriter, req *http.Request)
}

func NewService(repo repo.Repository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repo.Repository
}

func (s *service) UpdateMetric(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")
	rawValue := req.PathValue("value")
	switch typeMetrics {
	case "gauge":
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			http.Error(w, "invalid value", http.StatusBadRequest)
			return
		}
		err = s.updateGauge(name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "counter":
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			http.Error(w, "invalid value", http.StatusBadRequest)
			return
		}
		err = s.updateCounter(name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "invalid typeMetrics", http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return
}

func (s *service) updateGauge(name string, value float64) error {
	err := s.repo.SetGauge(name, value)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) updateCounter(name string, value int64) error {
	err := s.repo.SetCounter(name, value)
	if err != nil {
		return err
	}
	return nil
}
