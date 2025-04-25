package config

import (
	"flag"
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

	return cfg
}
