// Package config provides configuration management functionality for the application.
// It handles command-line flags and environment variables for both agent and server components.
package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type AgentConfig struct {
	MetricServerHost string
	SginKey          string
	RSAPublicKeyPath string
	ReportInterval   int
	PollInterval     int
	RateLimit        int
}
type PrioritiAgentConfig struct {
	MetricServerHost prioritized[string]
	SginKey          prioritized[string]
	RSAPublicKeyPath prioritized[string]
	ReportInterval   prioritized[int]
	PollInterval     prioritized[int]
	RateLimit        prioritized[int]
}

func (cfg *PrioritiAgentConfig) ToPlain() AgentConfig {
	return AgentConfig{
		PollInterval:     cfg.PollInterval.Get(),
		ReportInterval:   cfg.ReportInterval.Get(),
		MetricServerHost: cfg.MetricServerHost.Get(),
		SginKey:          cfg.SginKey.Get(),
		RSAPublicKeyPath: cfg.RSAPublicKeyPath.Get(),
		RateLimit:        cfg.RateLimit.Get(),
	}
}

func NewAgentConfig() *AgentConfig {
	cfg, err := ParseAgentFlagsArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

func DefaultConfig() *PrioritiAgentConfig {
	cfg := &PrioritiAgentConfig{}
	// Устанавливаем дефолты с приоритетом priorityDefault
	cfg.PollInterval.Set(2, priorityDefault)
	cfg.ReportInterval.Set(10, priorityDefault)
	cfg.MetricServerHost.Set("localhost:8080", priorityDefault)
	cfg.SginKey.Set("", priorityDefault)
	cfg.RateLimit.Set(1, priorityDefault)
	return cfg
}

// helper для десериализации из json в стандартный struct, потом копируем в prioritized
type rawAgentConfig struct {
	MetricServerHost string `json:"address"`
	SginKey          string `json:"sgin_key"`
	RSAPublicKeyPath string `json:"crypto_key"`
	ReportInterval   int    `json:"report_interval"`
	PollInterval     int    `json:"poll_interval"`
	RateLimit        int    `json:"rate_limit"`
}

// ParseAgentFlags читает конфиги из разных источников с расставлением приоритетов
func ParseAgentFlagsArgs(args []string) (*AgentConfig, error) {
	cfg := DefaultConfig()

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	pollFlag := fs.Int("p", cfg.PollInterval.Get(), "poll interval seconds")
	reportFlag := fs.Int("r", cfg.ReportInterval.Get(), "report interval seconds")
	addressFlag := fs.String("a", cfg.MetricServerHost.Get(), "server address")
	signKeyFlag := fs.String("k", cfg.SginKey.Get(), "sign key")
	rateLimitFlag := fs.Int("l", cfg.RateLimit.Get(), "rate limit for outgoing requests")
	cryptoKeyFlag := fs.String("crypto-key", "", "file public key path")
	configFileFlag := fs.String("config", "", "config json file path")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	var rawCfg rawAgentConfig

	// 1. Конфигурация из файла (флаг)
	if *configFileFlag != "" {
		f, err := os.Open(*configFileFlag)
		if err != nil {
			return nil, fmt.Errorf("open config file flag '%s' error: %w", *configFileFlag, err)
		}
		defer f.Close()
		err = parseJSONConfigFile(f, &rawCfg)
		if err != nil {
			return nil, err
		}
		setFromRaw(cfg, &rawCfg, priorityConfigFile)
	}

	// 2. Конфигурация из env файлу CONFIG_FILE (превалирует над конфигом из флага)
	if envFile, ok := os.LookupEnv("CONFIG_FILE"); ok {
		f, err := os.Open(envFile)
		if err != nil {
			return nil, fmt.Errorf("open config file env '%s' error: %w", envFile, err)
		}
		defer f.Close()
		err = parseJSONConfigFile(f, &rawCfg)
		if err != nil {
			return nil, err
		}
		setFromRaw(cfg, &rawCfg, priorityConfigFile)
	}

	// 3. Параметры из флагов (переопределяют значение если приоритет выше)
	cfg.PollInterval.Set(*pollFlag, priorityFlags)
	cfg.ReportInterval.Set(*reportFlag, priorityFlags)
	cfg.MetricServerHost.Set(*addressFlag, priorityFlags)
	cfg.SginKey.Set(*signKeyFlag, priorityFlags)
	cfg.RateLimit.Set(*rateLimitFlag, priorityFlags)
	if *cryptoKeyFlag != "" {
		cfg.RSAPublicKeyPath.Set(*cryptoKeyFlag, priorityFlags)
	}

	// 4. Переменные окружения всегда самый высокий приоритет
	if env, ok := os.LookupEnv("CRYPTO_KEY"); ok && env != "" {
		cfg.RSAPublicKeyPath.Set(env, priorityEnv)
	}
	if env, ok := os.LookupEnv("ADDRESS"); ok && env != "" {
		cfg.MetricServerHost.Set(env, priorityEnv)
	}
	if env, ok := os.LookupEnv("KEY"); ok && env != "" {
		cfg.SginKey.Set(env, priorityEnv)
	}

	// Парсим целочисленные переменные окружения напрямую
	if val, ok := os.LookupEnv("REPORT_INTERVAL"); ok && val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid REPORT_INTERVAL env %q: %w", val, err)
		}
		cfg.ReportInterval.Set(i, priorityEnv)
	}

	if val, ok := os.LookupEnv("POLL_INTERVAL"); ok && val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid POLL_INTERVAL env %q: %w", val, err)
		}
		cfg.PollInterval.Set(i, priorityEnv)
	}

	if val, ok := os.LookupEnv("RATE_LIMIT"); ok && val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT env %q: %w", val, err)
		}
		cfg.RateLimit.Set(i, priorityEnv)
	}
	// получаем конфиг агента отностильно всех приоритоев
	agentConfig := cfg.ToPlain()
	return &agentConfig, nil
}

func setFromRaw(cfg *PrioritiAgentConfig, raw *rawAgentConfig, priority int) {
	if raw.PollInterval != 0 {
		cfg.PollInterval.Set(raw.PollInterval, priority)
	}
	if raw.ReportInterval != 0 {
		cfg.ReportInterval.Set(raw.ReportInterval, priority)
	}
	if raw.MetricServerHost != "" {
		cfg.MetricServerHost.Set(raw.MetricServerHost, priority)
	}
	if raw.SginKey != "" {
		cfg.SginKey.Set(raw.SginKey, priority)
	}
	if raw.RSAPublicKeyPath != "" {
		cfg.RSAPublicKeyPath.Set(raw.RSAPublicKeyPath, priority)
	}
	if raw.RateLimit != 0 {
		cfg.RateLimit.Set(raw.RateLimit, priority)
	}
}
