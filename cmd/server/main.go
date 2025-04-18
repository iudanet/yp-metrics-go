package main

import (
	"net/http"

	"github.com/iudanet/yp-metrics-go/internal/repo"
	"github.com/iudanet/yp-metrics-go/internal/service"
)

func main() {
	repo := repo.NewStorage()
	svc := service.NewService(repo)

	m := http.NewServeMux()
	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)

	err := http.ListenAndServe(`localhost:8080`, m)
	if err != nil {
		panic(err)
	}
}
