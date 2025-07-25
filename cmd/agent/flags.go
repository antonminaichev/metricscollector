package main

import (
	"flag"

	"github.com/caarlos0/env"
)

// Config stores agent setting.
type Config struct {
	Address        string `env:"ADDRESS" envDefault:"localhost:8080"`
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"2"`
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"2"`
	RateLimit      int    `env:"RATE_LIMIT" envDefault:"30"`
	HashKey        string `env:"KEY"`
}

// NewConfig initialises new agent configuration.
func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	address := flag.String("a", cfg.Address, "{Host:port} for server")
	reportInterval := flag.Int("r", cfg.ReportInterval, "Report interval, seconds")
	pollInterval := flag.Int("p", cfg.PollInterval, "Poll interval, seconds")
	rateLimit := flag.Int("l", cfg.RateLimit, "Max concurrent requests")
	hashKey := flag.String("k", cfg.HashKey, "Hash key")
	flag.Parse()
	cfg.Address = *address
	cfg.ReportInterval = *reportInterval
	cfg.PollInterval = *pollInterval
	cfg.RateLimit = *rateLimit
	cfg.HashKey = *hashKey

	return cfg, nil
}
