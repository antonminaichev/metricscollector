package main

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	Address  string `env:"ADDRESS" envDefault:"localhost:8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"INFO"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	address := flag.String("a", cfg.Address, "{Host:port} for server")
	loglevel := flag.String("l", cfg.LogLevel, "Log level for server")
	flag.Parse()
	cfg.Address = *address
	cfg.LogLevel = *loglevel

	return cfg, nil
}
