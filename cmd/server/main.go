package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/server"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	pgStore "github.com/iudanet/yp-metrics-go/internal/storage/pg"
	"go.uber.org/zap"
)

var sugar zap.SugaredLogger

func main() {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	newLogger, err := logger.New("Info")
	if err != nil {
		log.Fatal(err)
	}

	// делаем регистратор SugaredLogger
	st := localStore.New()
	cfg := config.ParseServerFlags()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// восстановление базы из файла
	if cfg.Storage.Restore {
		err := st.LoadDB(ctx, cfg.Storage.Path)
		if err != nil {
			newLogger.Error("Failed to restore metrics", zap.Error(err))
		} else {
			newLogger.Info("Successfully restored metrics from disk")
		}
	}

	// Start the worker if store interval is greater than 0
	if cfg.Storage.StoreInterval > 0 {
		st.StartWorker(ctx, cfg.Storage, newLogger)
		newLogger.Info("Started metrics persistence worker",
			zap.Int("interval_seconds", cfg.Storage.StoreInterval))
	}
	svc := server.NewService(st, cfg, newLogger, st)
	if cfg.Storage.DatabaseDSN != "" {
		pg, err := pgStore.New(ctx, cfg.Storage.DatabaseDSN)
		if err != nil {
			newLogger.Error("Failed to connect to database", zap.Error(err))
			return
		}
		svc = server.NewService(pg, cfg, newLogger, pg)
	}

	// chi отключен для проходждения тестов. хотел сделать с нативным новым роутером.
	_ = chi.NewRouter()
	m := http.NewServeMux()
	m.Handle(`POST /update/{$}`, svc.CheckContentType(http.HandlerFunc(svc.UpdateMetricJSON)))
	m.Handle(`POST /updates/{$}`, svc.CheckContentType(svc.VerifyHash(http.HandlerFunc(svc.UpdateMetricsBatch))))
	m.Handle(`POST /value/{$}`, svc.CheckContentType(http.HandlerFunc(svc.GetMetricJSON)))

	m.HandleFunc(`POST /update/{typeMetrics}/{name}/{value}`, svc.UpdateMetric)
	m.HandleFunc(`GET /value/{typeMetrics}/{name}`, svc.GetMetric)
	m.HandleFunc(`GET /ping`, svc.Ping)
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
	st.WaitWorker()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		newLogger.Error("Server shutdown error", zap.Error(err))
	} else {
		newLogger.Info("Server gracefully stopped")
	}
}
