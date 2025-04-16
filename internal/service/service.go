package service

import (
	"net/http"
	"strconv"

	"github.com/iudanet/yp-metrics-go/internal/repo"
)

type Service interface {
	UpdateCounter(w http.ResponseWriter, req *http.Request)
	UpdateGauge(w http.ResponseWriter, req *http.Request)
}

func NewService(repo repo.Repository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repo.Repository
}

func (s *service) UpdateGauge(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	rawValue := req.PathValue("value")

	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	value, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		http.Error(w, "invalid value", http.StatusBadRequest)
		return
	}
	s.repo.SetGauge(name, value)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(make([]byte, 0))
}

func (s *service) UpdateCounter(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	rawValue := req.PathValue("value")
	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}
	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		http.Error(w, "invalid value", http.StatusBadRequest)
		return
	}
	s.repo.SetCounter(name, value)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(make([]byte, 0))
}
