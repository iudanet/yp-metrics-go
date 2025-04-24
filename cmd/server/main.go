package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
)

func main() {
	storage := storage.NewStorage()
	svc := server.NewService(storage)
	_ = chi.NewRouter()
	m := http.NewServeMux()
	// m.HandleFunc(`POST /updater/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)
	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)
	// m.HandleFunc(`POST /update/{typeMetrics}/{name}`, svc.UpdateMetric)

	m.HandleFunc(`GET /value/{typeMetrics}/{name}`, svc.GetMetric)
	m.HandleFunc(`GET /{$}`, svc.GetIndex)

	err := http.ListenAndServe(`localhost:8080`, m)
	if err != nil {
		panic(err)
	}
}
