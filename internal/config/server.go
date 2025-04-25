package config

import (
	"flag"
)

type ServerConfig struct {
	MetricServerHost string
}

func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{}

	flag.StringVar(&cfg.MetricServerHost, "a", "localhost:8080", "server address")
	flag.Parse()
	return cfg
}
