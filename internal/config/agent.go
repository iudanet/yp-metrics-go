package config

import (
	"flag"
	"time"
)

type AgentConfig struct {
	ReportInterval   time.Duration
	PollInterval     time.Duration
	MetricServerHost string
}

func NewAgentConfig() *AgentConfig {
	cfg := &AgentConfig{}

	flag.DurationVar(&cfg.PollInterval, "p", 2*time.Second, "poll interval")
	flag.DurationVar(&cfg.ReportInterval, "r", 10*time.Second, "report interval")
	flag.StringVar(&cfg.MetricServerHost, "a", "localhost:8080", "server address")
	flag.Parse()
	return cfg
}
