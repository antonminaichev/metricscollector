package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/caarlos0/env"
)

// Config stores server setting.
type Config struct {
	Address            string `env:"ADDRESS"`
	LogLevel           string `env:"LOG_LEVEL"`
	StoreInterval      int    `env:"STORE_INTERVAL"`
	FileStoragePath    string `env:"FILE_STORAGE_PATH"`
	Restore            bool   `env:"RESTORE"`
	DatabaseConnection string `env:"DATABASE_DSN"`
	HashKey            string `env:"KEY"`
	CryptoKey          string `env:"CRYPTO_KEY"`
}

func loadJSONConfig(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func pickConfigPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case strings.HasPrefix(a, "-c="):
			return strings.TrimPrefix(a, "-c=")
		}
	}
	return ""
}

// NewConfig initialises new server configuration.
func NewConfig() (*Config, error) {
	cfg := &Config{Address: "localhost:8080", LogLevel: "INFO", StoreInterval: 300, FileStoragePath: "./metrics/metrics.json", Restore: true}

	configPath := pickConfigPathFromArgs(os.Args[1:])
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	if configPath != "" {
		if err := loadJSONConfig(configPath, cfg); err != nil {
			return nil, err
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Определяем флаги
	address := flag.String("a", cfg.Address, "{Host:port} for server")
	loglevel := flag.String("l", cfg.LogLevel, "Log level for server")
	storeInterval := flag.Int("i", cfg.StoreInterval, "Store interval in seconds")
	filePath := flag.String("f", cfg.FileStoragePath, "File storage path")
	restore := flag.Bool("r", cfg.Restore, "Restore metrics from file")
	databaseConnection := flag.String("d", cfg.DatabaseConnection, "Database connection string")
	hashkey := flag.String("k", "", "Hash key")
	cryptoKey := flag.String("crypto-key", cfg.CryptoKey, "Path to private key")
	_ = flag.String("c", configPath, "Path to config file (JSON)")

	flag.Parse()

	// Обновляем конфигурацию значениями из флагов
	cfg.Address = *address
	cfg.LogLevel = *loglevel
	cfg.StoreInterval = *storeInterval
	cfg.FileStoragePath = *filePath
	cfg.Restore = *restore
	cfg.DatabaseConnection = *databaseConnection
	if cfg.HashKey == "" {
		cfg.HashKey = *hashkey
	}
	cfg.CryptoKey = *cryptoKey

	return cfg, nil
}
