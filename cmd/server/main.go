package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
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

	cfg := config.ParseServerFlags()
	ctx, cancel := context.WithCancel(context.Background())
	memWg := sync.WaitGroup{}
	defer cancel()
	var repo storage.Repository
	if cfg.Storage.DatabaseDSN != "" {
		repo, err = pgStore.New(ctx, cfg.Storage.DatabaseDSN)
		if err != nil {
			newLogger.Error("Failed to connect to database", zap.Error(err))
			return
		}

	} else {
		// если нет перменной для подключения к postgres используется локальное хранение
		st := localStore.New()

		if cfg.Storage.Restore {
			err := st.LoadDB(ctx, cfg.Storage.Path)
			if err != nil {
				newLogger.Error("Failed to restore metrics", zap.Error(err))
			} else {
				newLogger.Info("Successfully restored metrics from disk")
			}
		}
		if cfg.Storage.StoreInterval > 0 {
			memWg.Add(1)
			st.StartWorker(ctx, cfg.Storage, newLogger, &memWg)

			newLogger.Info("Started metrics persistence worker",
				zap.Int("interval_seconds", cfg.Storage.StoreInterval))
		}
		repo = st

	}
	svc := server.NewService(repo, cfg, newLogger, repo)

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
	if cfg.Storage.DatabaseDSN == "" {
		memWg.Wait()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		newLogger.Error("Server shutdown error", zap.Error(err))
	} else {
		newLogger.Info("Server gracefully stopped")
	}
}
