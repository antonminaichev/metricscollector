package main

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	Address        string `env:"ADDRESS" envDefault:"localhost:8080"`
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"2"`
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	address := flag.String("a", cfg.Address, "{Host:port} for server")
	reportInterval := flag.Int("r", cfg.ReportInterval, "Report interval, seconds")
	pollInterval := flag.Int("p", cfg.PollInterval, "Poll interval, seconds")
	flag.Parse()
	cfg.Address = *address
	cfg.ReportInterval = *reportInterval
	cfg.PollInterval = *pollInterval

	return cfg, nil
}
