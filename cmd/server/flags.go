package main

import (
	"flag"
	"os"

	"github.com/antonminaichev/metricscollector/internal/conf"
	"github.com/antonminaichev/metricscollector/internal/server"
	"github.com/caarlos0/env"
)

// NewConfig initialises new server configuration.
func NewConfig() (*server.Config, error) {
	cfg := &server.Config{Address: "localhost:8080", LogLevel: "INFO", StoreInterval: 300, FileStoragePath: "./metrics/metrics.json", Restore: true}

	configPath := conf.PickConfigPathFromArgs(os.Args[1:])
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	if configPath != "" {
		if err := conf.LoadJSONConfig(configPath, cfg); err != nil {
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
	trustedSubnet := flag.String("t", cfg.TrustedSubnet, "Trusted subnet in CIDR (e.g. 192.168.1.0/24)")
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
	cfg.TrustedSubnet = *trustedSubnet

	return cfg, nil
}
