package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"github.com/iudanet/yp-metrics-go/internal/agent"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/logger"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
	"go.uber.org/zap"
)

func main() {
	ctxStop, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	newLogger, err := logger.New("Info")
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.ParseAgentFlags()
	if err != nil {
		newLogger.Error("failed to parse agent flags", zap.Error(err))
		log.Fatal("failed to parse agent flags: ", err)
	}
	stor := localStore.New()

	a := agent.NewAgent(cfg, stor, newLogger)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.PollWorker(ctxStop)
	}()
	go a.StartWorkers(ctxStop, &wg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.ReportWorkerBatch(ctxStop)
	}()

	<-ctxStop.Done()
	newLogger.Info("Agent stopped")

	// Wait for all goroutines to finish
	wg.Wait()
	newLogger.Info("All workers have finished")
}
