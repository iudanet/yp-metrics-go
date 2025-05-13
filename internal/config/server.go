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
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		MetricServerHost: "localhost:8080",
		Storage: Storage{
			Restore:       false,
			Path:          "./db.json",
			StoreInterval: 300,
		},
	}
}

func ParseServerFlags() *ServerConfig {
	cfg := NewServerConfig()

	flag.StringVar(&cfg.MetricServerHost, "a", cfg.MetricServerHost, "server address. ENV: ADDRESS")
	flag.StringVar(&cfg.Storage.Path, "f", cfg.Storage.Path, "db file. ENV: FILE_STORAGE_PATH ")
	flag.IntVar(&cfg.Storage.StoreInterval, "i", cfg.Storage.StoreInterval, "Store Interval. ENV: STORE_INTERVAL")
	flag.BoolVar(&cfg.Storage.Restore, "r", cfg.Storage.Restore, "Restore from disk. Env: RESTORE")
	flag.Parse()

	envADDRESS := os.Getenv("ADDRESS")
	if envADDRESS != "" {
		cfg.MetricServerHost = envADDRESS
	}
	envFILESTORAGEPATH := os.Getenv("FILE_STORAGE_PATH")
	if envFILESTORAGEPATH != "" {
		cfg.Storage.Path = envFILESTORAGEPATH
	}

	envSTOREINTERVAL := os.Getenv("STORE_INTERVAL")
	if envSTOREINTERVAL != "" {
		interval, err := strconv.Atoi(envSTOREINTERVAL)
		if err != nil {
			log.Println("Error parsing ENV: STORE_INTERVAL", err)
		} else {
			cfg.Storage.StoreInterval = interval
		}
	}

	envRESTORE := os.Getenv("RESTORE")
	if envRESTORE != "" {
		restore, err := strconv.ParseBool(envRESTORE)
		if err != nil {
			log.Println("Error parsing ENV: RESTORE", err)
		} else {
			cfg.Storage.Restore = restore
		}
	}

	return cfg
}
