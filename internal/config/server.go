package config

import (
	"flag"
	"os"
)

type ServerConfig struct {
	MetricServerHost string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		MetricServerHost: "localhost:8080",
	}
}

func ParseServerFlags() *ServerConfig {
	cfg := NewServerConfig()

	flag.StringVar(&cfg.MetricServerHost, "a", cfg.MetricServerHost, "server address")
	flag.Parse()
	envADDRESS := os.Getenv("ADDRESS")
	if envADDRESS != "" {
		cfg.MetricServerHost = envADDRESS
	}

	return cfg
}
