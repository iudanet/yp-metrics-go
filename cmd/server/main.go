package main

import (
	"net/http"

	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
)

func main() {
	storage := storage.NewStorage()
	svc := server.NewService(storage)

	m := http.NewServeMux()
	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)

	err := http.ListenAndServe(`localhost:8080`, m)
	if err != nil {
		panic(err)
	}
}
