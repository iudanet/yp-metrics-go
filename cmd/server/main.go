package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"go.uber.org/zap"
)

var sugar zap.SugaredLogger

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	newLogger, err := logger.New("Info")
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}

	// делаем регистратор SugaredLogger
	storage := storage.NewStorage()
	cfg := config.ParseServerFlags()

	// восстановление базы из файла
	if cfg.Storage.Restore {
		err := storage.LoadDB(cfg.Storage.Path)
		if err != nil {
			newLogger.Error("Failed to restore metrics", zap.Error(err))
		} else {
			newLogger.Info("Successfully restored metrics from disk")
		}
	}
	// Start the worker if store interval is greater than 0
	if cfg.Storage.StoreInterval > 0 {
		storage.StartWorker(ctx, cfg.Storage, newLogger)
		newLogger.Info("Started metrics persistence worker",
			zap.Int("interval_seconds", cfg.Storage.StoreInterval))
	}

	svc := server.NewService(storage, cfg, newLogger)
	// chi отключен для проходждения тестов. хотел сделать с нативным новым роутером.
	_ = chi.NewRouter()
	m := http.NewServeMux()
	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)
	m.HandleFunc(`POST /update/{$}`, svc.UpdateMetricJSON)
	m.HandleFunc(`GET /value/{typeMetrics}/{name}`, svc.GetMetric)
	m.HandleFunc(`POST /value/{$}`, svc.GetMetricJSON)
	m.HandleFunc(`GET /{$}`, svc.GetIndex)

	srv := &http.Server{
		Addr:    cfg.MetricServerHost,
		Handler: svc.GzipMiddleware(svc.WithLogging(m)),
	}

	go func() {
		newLogger.Info("Running server", zap.String("address", cfg.MetricServerHost))
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			newLogger.Error("Server error", zap.Error(err))
			cancel()
		}
	}()

	sig := <-sigCh
	newLogger.Info("Received signal", zap.String("signal", sig.String()))
	cancel()
	// ждем пока сохранится база при отключении
	storage.WaitWorker()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		newLogger.Error("Server shutdown error", zap.Error(err))
	} else {
		newLogger.Info("Server gracefully stopped")
	}
}
