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
	cfg := &server.Config{
		Address:         "localhost:8080",
		GRPCAddress:     "localhost:9090",
		LogLevel:        "INFO",
		StoreInterval:   300,
		FileStoragePath: "./metrics/metrics.json",
		Restore:         true,
		Mode:            "http",
	}

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

	// Флаги
	address := flag.String("a", cfg.Address, "{Host:port} for HTTP server")
	grpcAddr := flag.String("grpc-addr", cfg.GRPCAddress, "{Host:port} for gRPC server")
	loglevel := flag.String("l", cfg.LogLevel, "Log level for server")
	storeInterval := flag.Int("i", cfg.StoreInterval, "Store interval in seconds")
	filePath := flag.String("f", cfg.FileStoragePath, "File storage path")
	restore := flag.Bool("r", cfg.Restore, "Restore metrics from file")
	databaseConnection := flag.String("d", cfg.DatabaseConnection, "Database connection string")
	hashkey := flag.String("k", cfg.HashKey, "Hash key")
	cryptoKey := flag.String("crypto-key", cfg.CryptoKey, "Path to private key")
	trustedSubnet := flag.String("t", cfg.TrustedSubnet, "Trusted subnet in CIDR (e.g. 192.168.1.0/24)")
	mode := flag.String("mode", cfg.Mode, "Mode: http|grpc")
	_ = flag.String("c", configPath, "Path to config file (JSON)")

	flag.Parse()

	// Обновляем конфигурацию значениями из флагов
	cfg.Address = *address
	cfg.GRPCAddress = *grpcAddr
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
	cfg.Mode = *mode

	return cfg, nil
}
