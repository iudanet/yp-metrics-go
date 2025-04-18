package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/agent"
	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/storage"
)

func main() {
	ctxCancel, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctxStop, stop := signal.NotifyContext(ctxCancel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	pollInterval := time.Duration(2)
	reportInterval := time.Duration(10)
	metricServerAddress := "localhost:8080"

	cfg := config.NewAdminConfig(pollInterval, reportInterval, metricServerAddress)
	stor := storage.NewStorage()

	a := agent.NewAgent(cfg, stor)
	go a.PollWorker()
	go a.ReportWorker()

	select {
	case <-ctxStop.Done():
		log.Println("Agent stopped")
	case <-ctxCancel.Done():
		log.Println("Agent canceled")
	}
}
