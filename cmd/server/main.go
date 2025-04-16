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
	m.HandleFunc(`POST /update/counter/{name}/{value}`, svc.UpdateCounter)
	m.HandleFunc(`POST /update/gauge/{name}/{value}`, svc.UpdateGauge)

	err := http.ListenAndServe(`localhost:8080`, m)
	if err != nil {
		panic(err)
	}
}
