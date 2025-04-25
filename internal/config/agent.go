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
	return &AgentConfig{
		PollInterval:     2 * time.Second,
		ReportInterval:   10 * time.Second,
		MetricServerHost: "localhost:8080",
	}
}

func ParseAgentFlags() *AgentConfig {
	cfg := NewAgentConfig()

	pollInterval := flag.Int("p", 2, "poll interval seconds")
	reportInterval := flag.Int("r", 10, "report interval seconds")
	flag.StringVar(&cfg.MetricServerHost, "a", cfg.MetricServerHost, "server address")

	flag.Parse()

	cfg.PollInterval = time.Duration(*pollInterval) * time.Second
	cfg.ReportInterval = time.Duration(*reportInterval) * time.Second

	return cfg
}
