package config

import (
	"time"
)

type AdminConfig struct {
	ReportInterval   time.Duration
	PollInterval     time.Duration
	MetricServerHost string
}

func NewAdminConfig(pollInterval, reportInterval time.Duration, server string) *AdminConfig {
	return &AdminConfig{
		PollInterval:     pollInterval,
		ReportInterval:   reportInterval,
		MetricServerHost: server,
	}
}
