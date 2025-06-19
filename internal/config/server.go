package config

import (
	"flag"
	"log"
	"os"
	"strconv"
)

type ServerConfig struct {
	MetricServerHost string
	Storage          Storage
}

type Storage struct {
	Restore       bool
	Path          string
	StoreInterval int
	DatabaseDSN   string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		MetricServerHost: "localhost:8080",
		Storage: Storage{
			Restore:       false,
			Path:          "/tmp/metrics-db.json",
			StoreInterval: 300,
			DatabaseDSN:   "",
		},
	}
}

func ParseServerFlags() *ServerConfig {
	cfg := NewServerConfig()

	flag.StringVar(&cfg.MetricServerHost, "a", cfg.MetricServerHost, "server address. ENV: ADDRESS")
	flag.StringVar(&cfg.Storage.Path, "f", cfg.Storage.Path, "db file. ENV: FILE_STORAGE_PATH ")
	flag.StringVar(&cfg.Storage.DatabaseDSN, "d", cfg.Storage.DatabaseDSN, "Postgres DSN uri postgres://username:password@localhost:5432/metrics_db. ENV: DATABASE_DSN ")
	flag.IntVar(&cfg.Storage.StoreInterval, "i", cfg.Storage.StoreInterval, "Store Interval. ENV: STORE_INTERVAL")
	flag.BoolVar(&cfg.Storage.Restore, "r", cfg.Storage.Restore, "Restore from disk. Env: RESTORE")
	flag.Parse()

	envAddress := os.Getenv("ADDRESS")
	if envAddress != "" {
		cfg.MetricServerHost = envAddress
	}
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if envFileStoragePath != "" {
		cfg.Storage.Path = envFileStoragePath
	}
	envDatabaseDSN := os.Getenv("DATABASE_DSN")
	if envDatabaseDSN != "" {
		cfg.Storage.DatabaseDSN = envDatabaseDSN
	}
	envStoreInterval := os.Getenv("STORE_INTERVAL")
	if envStoreInterval != "" {
		interval, err := strconv.Atoi(envStoreInterval)
		if err != nil {
			log.Println("Error parsing ENV: STORE_INTERVAL", err)
		} else {
			cfg.Storage.StoreInterval = interval
		}
	}

	envRestore := os.Getenv("RESTORE")
	if envRestore != "" {
		restore, err := strconv.ParseBool(envRestore)
		if err != nil {
			log.Println("Error parsing ENV: RESTORE", err)
		} else {
			cfg.Storage.Restore = restore
		}
	}

	return cfg
}
