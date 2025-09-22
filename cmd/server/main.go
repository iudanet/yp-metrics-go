package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/iudanet/yp-metrics-go/api/grpc/grpcmetrics"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	"github.com/iudanet/yp-metrics-go/internal/server"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	pgStore "github.com/iudanet/yp-metrics-go/internal/storage/pg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	sugar        zap.SugaredLogger
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
	// делаем регистратор SugaredLogger
	newLogger, err := logger.New("Info")
	if err != nil {
		log.Fatal(err)
	}

	cfg := config.NewServerConfig()

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
			err = st.LoadDB(ctx, cfg.Storage.Path)
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

	srv := &http.Server{
		Addr:    cfg.MetricServerHost,
		Handler: svc.GetHandlerWithMiddleware(),
	}

	go func() {
		newLogger.Info("Running server", zap.String("address", cfg.MetricServerHost))
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			newLogger.Error("Server error", zap.Error(err))

			cancel()
		}
	}()

	grpcLis, err := net.Listen("tcp", cfg.GRPCAddress) // порт из конфига
	if err != nil {
		newLogger.Fatal("Failed to listen on GRPC address", zap.String("address", cfg.GRPCAddress), zap.Error(err))
	}
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		svc.IPVerificationInterceptor(),
		svc.LoggingInterceptor(),
	))
	grpcmetrics.RegisterMetricsServiceServer(grpcServer, server.NewGRPCServer(repo, newLogger))

	go func() {
		newLogger.Info("Starting GRPC server", zap.String("address", cfg.GRPCAddress))
		if serveErr := grpcServer.Serve(grpcLis); serveErr != nil && serveErr != grpc.ErrServerStopped {
			newLogger.Error("GRPC serve failed", zap.Error(serveErr))
		}
	}()

	// Graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-signalCh
	newLogger.Info("Received signal", zap.String("signal", sig.String()))

	cancel()
	// ждем пока сохранится база при отключении
	if cfg.Storage.DatabaseDSN == "" {
		memWg.Wait()
	}

	// Graceful shutdown for HTTP server
	shutdCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdCtx)
	if err != nil {
		newLogger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		newLogger.Info("HTTP server gracefully stopped")
	}

	// Graceful shutdown for GRPC server with timeout
	grpcShutdownCtx, grpcShutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer grpcShutdownCancel()

	grpcShutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcShutdownDone)
	}()

	select {
	case <-grpcShutdownDone:
		newLogger.Info("GRPC server gracefully stopped")
	case <-grpcShutdownCtx.Done():
		newLogger.Warn("GRPC server shutdown timed out, forcing stop")
		grpcServer.Stop()
	}
}
