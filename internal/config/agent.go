package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type AgentConfig struct {
	ReportInterval   int
	PollInterval     int
	MetricServerHost string
}

func NewAgentConfig() *AgentConfig {
	return &AgentConfig{
		PollInterval:     2,
		ReportInterval:   10,
		MetricServerHost: "localhost:8080",
	}
}

func ParseAgentFlags() (*AgentConfig, error) {
	cfg := NewAgentConfig()

	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval seconds")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval seconds")
	flag.StringVar(&cfg.MetricServerHost, "a", cfg.MetricServerHost, "server address")

	flag.Parse()

	envADDRESS := os.Getenv("ADDRESS")
	if envADDRESS != "" {
		cfg.MetricServerHost = envADDRESS
	}
	envReportInterval := os.Getenv("REPORT_INTERVAL")
	if envReportInterval != "" {
		r, err := strconv.Atoi(envReportInterval)
		if err != nil {
			fmt.Println("Ошибка env REPORT_INTERVAL:", err)
			return nil, err
		}

		cfg.ReportInterval = r
	}

	envPollInterval := os.Getenv("POLL_INTERVAL")
	if envPollInterval != "" {
		p, err := strconv.Atoi(envPollInterval)
		if err != nil {
			fmt.Println("Ошибка env POLL_INTERVAL:", err)
			return nil, err
		}
		cfg.PollInterval = p
	}

	return cfg, nil
}
