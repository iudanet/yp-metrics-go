package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iudanet/yp-metrics-go/internal/agent"
	"github.com/iudanet/yp-metrics-go/internal/config"
	localStore "github.com/iudanet/yp-metrics-go/internal/storage/local"
)

func main() {
	ctxCancel, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctxStop, stop := signal.NotifyContext(ctxCancel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	cfg, err := config.ParseAgentFlags()
	if err != nil {
		log.Printf("failed to parse agent flags: %v", err)
		os.Exit(1)
	}
	stor := localStore.New()

	a := agent.NewAgent(cfg, stor)
	go a.PollWorker(ctxStop)
	go a.ReportWorker(ctxStop)
	// go a.ReportWorkerBatch(ctxStop)

	select {
	case <-ctxStop.Done():
		log.Println("Agent stopped")
	case <-ctxCancel.Done():
		log.Println("Agent canceled")
	}
}
