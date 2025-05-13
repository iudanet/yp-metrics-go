package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"go.uber.org/zap"
)

var sugar zap.SugaredLogger

func main() {
	newLogger, err := logger.New("Info")
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}

	// делаем регистратор SugaredLogger
	storage := storage.NewStorage()
	cfg := config.ParseServerFlags()
	svc := server.NewService(storage, cfg, newLogger)
	// chi отключен для проходждения тестов. хотел сделать с нативным новым роутером.
	_ = chi.NewRouter()
	m := http.NewServeMux()
	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)
	m.HandleFunc(`POST /update/{$}`, svc.UpdateMetricJSON)
	m.HandleFunc(`GET /value/{typeMetrics}/{name}`, svc.GetMetric)
	m.HandleFunc(`POST /value/{$}`, svc.GetMetricJSON)
	m.HandleFunc(`GET /{$}`, svc.GetIndex)
	newLogger.Info("Running server", zap.String("address", cfg.MetricServerHost))
	err = http.ListenAndServe(cfg.MetricServerHost, svc.WithLogging(m))
	if err != nil {
		panic(err)
	}
}
