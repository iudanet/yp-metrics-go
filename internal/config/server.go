package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	MetricServerHost  string
	SginKey           string
	RSAPrivateKeyPath string
	Storage           Storage
}

type Storage struct {
	Path          string
	DatabaseDSN   string
	StoreInterval int // в секундах
	Restore       bool
}

// структура с приоритетными обертками
type PrioritiServerConfig struct {
	MetricServerHost  prioritized[string]
	SginKey           prioritized[string]
	RSAPrivateKeyPath prioritized[string]

	StoragePath          prioritized[string]
	StorageDatabaseDSN   prioritized[string]
	StorageStoreInterval prioritized[int]
	StorageRestore       prioritized[bool]
}

// raw структура для JSON конфига
type rawServerConfig struct {
	Address       string `json:"address"`
	SginKey       string `json:"sgin_key"`
	CryptoKey     string `json:"crypto_key"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	StoreInterval string `json:"store_interval"`
	Restore       bool   `json:"restore"`
}

func DefaultServerConfig() *PrioritiServerConfig {
	cfg := &PrioritiServerConfig{}

	cfg.MetricServerHost.Set("localhost:8080", priorityDefault)
	cfg.SginKey.Set("", priorityDefault)
	cfg.RSAPrivateKeyPath.Set("", priorityDefault)

	cfg.StoragePath.Set("/tmp/metrics-db.json", priorityDefault)
	cfg.StorageDatabaseDSN.Set("", priorityDefault)
	cfg.StorageStoreInterval.Set(300, priorityDefault) // 5 минут по умолчанию
	cfg.StorageRestore.Set(false, priorityDefault)

	return cfg
}

func (c *PrioritiServerConfig) ToPlain() *ServerConfig {
	return &ServerConfig{
		MetricServerHost:  c.MetricServerHost.Get(),
		SginKey:           c.SginKey.Get(),
		RSAPrivateKeyPath: c.RSAPrivateKeyPath.Get(),
		Storage: Storage{
			Path:          c.StoragePath.Get(),
			DatabaseDSN:   c.StorageDatabaseDSN.Get(),
			StoreInterval: c.StorageStoreInterval.Get(),
			Restore:       c.StorageRestore.Get(),
		},
	}
}

func NewServerConfig() *ServerConfig {
	cfg, err := ParseServerFlagsArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

func ParseServerFlagsArgs(args []string) (*ServerConfig, error) {
	cfg := DefaultServerConfig()

	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	configFileFlag := fs.String("config", "", "config JSON file path")

	fs.StringVar(&cfg.MetricServerHost.value, "a", cfg.MetricServerHost.Get(), "server address. ENV: ADDRESS")
	fs.StringVar(&cfg.StoragePath.value, "f", cfg.StoragePath.Get(), "db file. ENV: FILE_STORAGE_PATH")
	fs.StringVar(&cfg.StorageDatabaseDSN.value, "d", cfg.StorageDatabaseDSN.Get(), "Postgres DSN uri. ENV: DATABASE_DSN")
	fs.StringVar(&cfg.SginKey.value, "k", cfg.SginKey.Get(), "sign key. ENV: KEY")
	fs.IntVar(&cfg.StorageStoreInterval.value, "i", cfg.StorageStoreInterval.Get(), "Store Interval in seconds. ENV: STORE_INTERVAL")
	fs.BoolVar(&cfg.StorageRestore.value, "r", cfg.StorageRestore.Get(), "Restore from disk. ENV: RESTORE")
	fs.StringVar(&cfg.RSAPrivateKeyPath.value, "crypto-key", cfg.RSAPrivateKeyPath.Get(), "File Private Key. ENV: CRYPTO_KEY")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Устанавливаем приоритет 2 для значений из флагов
	cfg.MetricServerHost.priority = priorityFlags
	cfg.StoragePath.priority = priorityFlags
	cfg.StorageDatabaseDSN.priority = priorityFlags
	cfg.SginKey.priority = priorityFlags
	cfg.StorageStoreInterval.priority = priorityFlags
	cfg.StorageRestore.priority = priorityFlags
	cfg.RSAPrivateKeyPath.priority = priorityFlags

	// 1. Конфиг из файла, по пути через флаг -config (priorityConfigFile)
	if *configFileFlag != "" {
		f, err := os.Open(*configFileFlag)
		if err != nil {
			return nil, fmt.Errorf("open config file flag '%s' error: %w", *configFileFlag, err)
		}
		defer f.Close()

		var raw rawServerConfig
		if err := parseJSONConfigFile(f, &raw); err != nil {
			return nil, err
		}
		if err := setFromRawServerConfig(cfg, &raw, priorityConfigFile); err != nil {
			return nil, err
		}
	}

	// 2. Конфиг из env-переменной CONFIG_FILE (превалирует над конфигом из флага)
	if envFile, ok := os.LookupEnv("CONFIG_FILE"); ok {
		f, err := os.Open(envFile)
		if err != nil {
			return nil, fmt.Errorf("open config file env '%s' error: %w", envFile, err)
		}
		defer f.Close()

		var raw rawServerConfig
		if err := parseJSONConfigFile(f, &raw); err != nil {
			return nil, err
		}
		if err := setFromRawServerConfig(cfg, &raw, priorityConfigFile); err != nil {
			return nil, err
		}
	}

	// 3. Переменные окружения (приоритет 1)
	if env, ok := os.LookupEnv("CRYPTO_KEY"); ok && env != "" {
		cfg.RSAPrivateKeyPath.Set(env, priorityEnv)
	}
	if env, ok := os.LookupEnv("ADDRESS"); ok && env != "" {
		cfg.MetricServerHost.Set(env, priorityEnv)
	}
	if env, ok := os.LookupEnv("KEY"); ok && env != "" {
		cfg.SginKey.Set(env, priorityEnv)
	}
	if env, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && env != "" {
		cfg.StoragePath.Set(env, priorityEnv)
	}
	if env, ok := os.LookupEnv("DATABASE_DSN"); ok && env != "" {
		cfg.StorageDatabaseDSN.Set(env, priorityEnv)
	}

	if val, ok := os.LookupEnv("STORE_INTERVAL"); ok && val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid STORE_INTERVAL env %q: %w", val, err)
		}
		cfg.StorageStoreInterval.Set(i, priorityEnv)
	}

	if val, ok := os.LookupEnv("RESTORE"); ok && val != "" {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid RESTORE env %q: %w", val, err)
		}
		cfg.StorageRestore.Set(b, priorityEnv)
	}

	return cfg.ToPlain(), nil
}

func setFromRawServerConfig(cfg *PrioritiServerConfig, raw *rawServerConfig, priority int) error {
	if raw.Address != "" {
		cfg.MetricServerHost.Set(raw.Address, priority)
	}
	if raw.SginKey != "" {
		cfg.SginKey.Set(raw.SginKey, priority)
	}
	if raw.CryptoKey != "" {
		cfg.RSAPrivateKeyPath.Set(raw.CryptoKey, priority)
	}
	if raw.StoreFile != "" {
		cfg.StoragePath.Set(raw.StoreFile, priority)
	}
	if raw.DatabaseDSN != "" {
		cfg.StorageDatabaseDSN.Set(raw.DatabaseDSN, priority)
	}
	cfg.StorageRestore.Set(raw.Restore, priority)

	// Разбираем store_interval строку (например "1s")
	if raw.StoreInterval != "" {
		dur, err := time.ParseDuration(raw.StoreInterval)
		if err != nil {
			return fmt.Errorf("invalid store_interval in config: %w", err)
		}
		seconds := int(dur.Seconds())
		cfg.StorageStoreInterval.Set(seconds, priority)
	}

	return nil
}
