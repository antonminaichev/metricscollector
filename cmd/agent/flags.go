package main

import (
	"flag"
	"os"

	"github.com/antonminaichev/metricscollector/internal/agent"
	"github.com/antonminaichev/metricscollector/internal/conf"
	"github.com/caarlos0/env"
)

// NewConfig initialises new agent configuration.
func NewConfig() (*agent.Config, error) {
	cfg := &agent.Config{
		Address:        "localhost:8080",
		GRPCAddress:    "localhost:9090",
		ReportInterval: 2,
		PollInterval:   2,
		RateLimit:      30,
		Mode:           "http",
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

	address := flag.String("a", cfg.Address, "{Host:port} for HTTP server")
	grpcAddr := flag.String("grpc-addr", cfg.GRPCAddress, "{Host:port} for gRPC server")
	reportInterval := flag.Int("r", cfg.ReportInterval, "Report interval, seconds")
	pollInterval := flag.Int("p", cfg.PollInterval, "Poll interval, seconds")
	rateLimit := flag.Int("l", cfg.RateLimit, "Max concurrent requests (HTTP)")
	hashKey := flag.String("k", cfg.HashKey, "Hash key")
	cryptoKey := flag.String("crypto-key", cfg.CryptoKey, "Path to public key")
	mode := flag.String("mode", cfg.Mode, "Transport mode: http|grpc")
	_ = flag.String("c", configPath, "Path to config file (JSON)")

	flag.Parse()

	cfg.Address = *address
	cfg.GRPCAddress = *grpcAddr
	cfg.ReportInterval = *reportInterval
	cfg.PollInterval = *pollInterval
	cfg.RateLimit = *rateLimit
	cfg.HashKey = *hashKey
	cfg.CryptoKey = *cryptoKey
	cfg.Mode = *mode
	return cfg, nil
}
