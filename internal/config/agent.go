package config

import (
	"flag"
	"log"
	"os"
	"strconv"
)

type AgentConfig struct {
	ReportInterval   int
	PollInterval     int
	MetricServerHost string
	SginKey          string
	RateLimit        int
}

func NewAgentConfig() *AgentConfig {
	return &AgentConfig{
		PollInterval:     2,
		ReportInterval:   10,
		MetricServerHost: "localhost:8080",
		SginKey:          "",
		RateLimit:        1,
	}
}

func ParseAgentFlags() (*AgentConfig, error) {
	cfg := NewAgentConfig()

	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval seconds")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval seconds")
	flag.StringVar(&cfg.MetricServerHost, "a", cfg.MetricServerHost, "server address")
	flag.StringVar(&cfg.SginKey, "k", cfg.SginKey, "Sgin key")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit for outgoing requests")

	flag.Parse()

	envADDRESS := os.Getenv("ADDRESS")
	if envADDRESS != "" {
		cfg.MetricServerHost = envADDRESS
	}
	envKEY := os.Getenv("KEY")
	if envKEY != "" {
		cfg.SginKey = envKEY
	}
	envReportInterval := os.Getenv("REPORT_INTERVAL")
	if envReportInterval != "" {
		r, err := strconv.Atoi(envReportInterval)
		if err != nil {
			log.Println("Ошибка env REPORT_INTERVAL:", err)
			return nil, err
		}

		cfg.ReportInterval = r
	}

	envPollInterval := os.Getenv("POLL_INTERVAL")
	if envPollInterval != "" {
		p, err := strconv.Atoi(envPollInterval)
		if err != nil {
			log.Println("Ошибка env POLL_INTERVAL:", err)
			return nil, err
		}
		cfg.PollInterval = p
	}
	envRateLimit := os.Getenv("RATE_LIMIT")
	if envRateLimit != "" {
		rl, err := strconv.Atoi(envRateLimit)
		if err != nil {
			log.Println("Ошибка env RATE_LIMIT:", err)
			return nil, err
		}
		cfg.RateLimit = rl
	}
	return cfg, nil
}
